/*
Copyright 2021 The KubeOne Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cilium

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// daemonSet installs the calico/node container, as well as the Calico CNI plugins and network config on each
// master and worker node in a Kubernetes cluster
func daemonSet(installCNIImage, ciliumImage string) *appsv1.DaemonSet {
	maxUnavailable := intstr.FromInt(1)
	terminationGracePeriodSeconds := int64(1)
	privileged := true
	optional := true
	mountPropagation := corev1.MountPropagationHostToContainer
	hostPathDirectoryOrCreate := corev1.HostPathDirectoryOrCreate
	defaultMode := int32(420)

	commonLabels := map[string]string{
		"k8s-app": "cilium",
	}

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cilium",
			Namespace: metav1.NamespaceSystem,
			Labels:    commonLabels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: commonLabels,
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: commonLabels,
					Annotations: map[string]string{
						// This annotation plus the CriticalAddonsOnly toleration makes
						// cilium to be a critical pod in the cluster, which ensures cilium
						// gets priority scheduling.
						// https://kubernetes.io/docs/tasks/administer-cluster/guaranteed-scheduling-critical-addon-pods/
						"scheduler.alpha.kubernetes.io/critical-pod": "",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:       "cilium",
					DeprecatedServiceAccount: "cilium",
					PriorityClassName:        "system-node-critical",
					HostNetwork:              true,
					RestartPolicy:            corev1.RestartPolicyAlways,
					// Minimize downtime during a rolling upgrade or deletion; tell Kubernetes to do a "force
					// deletion": https://kubernetes.io/docs/concepts/workloads/pods/pod/#termination-of-pods
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/os",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"linux"},
											},
										},
									},
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "beta.kubernetes.io/os",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"linux"},
											},
										},
									},
								},
							},
						},
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									TopologyKey: corev1.LabelHostname,
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "k8s-app",
												Operator: metav1.LabelSelectorOpIn,
												Values:   []string{"cilium"},
											},
										},
									},
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists,
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:            "clean-cilium-state",
							Image:           installCNIImage,
							Command:         []string{"/init-container.sh"},
							ImagePullPolicy: corev1.PullAlways,
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"NET_ADMIN"},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "CILIUM_ALL_STATE",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											Key: "clean-cilium-state",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "cilium-config",
											},
											Optional: &optional,
										},
									},
								},
								{
									Name: "CILIUM_BPF_STATE",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											Key: "clean-cilium-bpf-state",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "cilium-config",
											},
											Optional: &optional,
										},
									},
								},
								{
									Name: "CILIUM_WAIT_BPF_MOUNT",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											Key: "wait-bpf-mount",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "cilium-config",
											},
											Optional: &optional,
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:             "bpf-maps",
									MountPath:        "/sys/fs/bpf",
									MountPropagation: &mountPropagation,
								},
								{
									Name:      "cilium-run",
									MountPath: "/var/run/cilium",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "cilium-agent",
							Image:           ciliumImage,
							ImagePullPolicy: corev1.PullAlways,
							Command:         []string{"cilium-agent"},
							Args:            []string{"--config-dir=/tmp/cilium/config-map"},
							Env: []corev1.EnvVar{
								{
									Name: "K8S_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "spec.nodeName",
										},
									},
								},
								{
									Name: "CILIUM_K8S_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.namespace",
										},
									},
								},
								{
									Name: "CILIUM_FLANNEL_MASTER_DEVICE",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											Key: "flannel-master-device",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "cilium-config",
											},
											Optional: &optional,
										},
									},
								},
								{
									Name: "CILIUM_FLANNEL_UNINSTALL_ON_EXIT",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											Key: "flannel-uninstall-on-exit",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "cilium-config",
											},
											Optional: &optional,
										},
									},
								},
								{
									Name:  "CILIUM_CLUSTERMESH_CONFIG",
									Value: "/var/lib/cilium/clustermesh/",
								},
								{
									Name: "CILIUM_CNI_CHAINING_MODE",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											Key: "cni-chaining-mode",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "cilium-config",
											},
											Optional: &optional,
										},
									},
								},
								{
									Name: "CILIUM_CUSTOM_CNI_CONF",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											Key: "custom-cni-conf",
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "cilium-config",
											},
											Optional: &optional,
										},
									},
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"NET_ADMIN",
										"SYS_MODULE",
									},
								},
							},
							Lifecycle: &corev1.Lifecycle{
								PostStart: &corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/cni-install.sh",
											"--enable-debug=false",
											"--cni-exclusive=true",
										},
									},
								},
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{"/cni-uninstall.sh"},
									},
								},
							},
							LivenessProbe: &corev1.Probe{
								FailureThreshold: 10,
								// The initial delay for the liveness probe is intentionally large to
								// avoid an endless kill & restart cycle if in the event that the initial
								// bootstrapping takes longer than expected.
								// Starting from Kubernetes 1.20, we are using startupProbe instead
								// of this field.
								InitialDelaySeconds: 120,
								PeriodSeconds:       30,
								SuccessThreshold:    1,
								TimeoutSeconds:      5,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(9876),
										Host:   "127.0.0.1",
										Scheme: corev1.URISchemeHTTP,
										HTTPHeaders: []corev1.HTTPHeader{
											{
												Name:  "brief",
												Value: "true",
											},
										},
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								FailureThreshold:    3,
								InitialDelaySeconds: 5,
								PeriodSeconds:       30,
								SuccessThreshold:    1,
								TimeoutSeconds:      5,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(9876),
										Host:   "127.0.0.1",
										Scheme: corev1.URISchemeHTTP,
										HTTPHeaders: []corev1.HTTPHeader{
											{
												Name:  "brief",
												Value: "true",
											},
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/sys/fs/bpf",
									Name:      "bpf-maps",
								},
								{
									MountPath: "/var/run/cilium",
									Name:      "cilium-run",
								},
								{
									MountPath: "/host/opt/cni/bin",
									Name:      "cni-path",
								},
								{
									MountPath: "/host/etc/cni/net.d",
									Name:      "etc-cni-netd",
								},
								{
									MountPath: "/var/lib/cilium/clustermesh",
									Name:      "clustermesh-secrets",
									ReadOnly:  true,
								},
								{
									MountPath: "/tmp/cilium/config-map",
									Name:      "cilium-config-path",
									ReadOnly:  true,
								},

								// Needed to be able to load kernel modules
								{
									MountPath: "/lib/modules",
									Name:      "lib-modules",
									ReadOnly:  true,
								},
								{
									MountPath: "/run/xtables.lock",
									Name:      "xtables-lock",
								},
								{
									MountPath: "/var/lib/cilium/tls/hubble",
									Name:      "hubble-tls",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						// To keep state between restarts / upgrades
						{
							Name: "cilium-run",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/run/cilium",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						// To keep state between restarts / upgrades for bpf maps
						{
							Name: "bpf-maps",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sys/fs/bpf",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						// To install cilium cni plugin in the host
						{
							Name: "cni-path",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/opt/cni/bin",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						// To install cilium cni configuration in the host
						{
							Name: "etc-cni-netd",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/cni/net.d",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						// To be able to load kernel modules
						{
							Name: "lib-modules",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/lib/modules",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						// To access iptables concurrently with other processes (e.g. kube-proxy)
						{
							Name: "xtables-lock",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run/xtables.lock",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						// To read the clustermesh configuration
						{
							Name: "clustermesh-secrets",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  "cilium-clustermesh",
									DefaultMode: &defaultMode,
									Optional:    &optional,
								},
							},
						},
						// To read the configuration from the config map
						{
							Name: "cilium-config-path",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "cilium-config",
									},
								},
							},
						},
						// Hubble-TLS
						{
							Name: "hubble-tls",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									Sources: []corev1.VolumeProjection{
										{
											Secret: &corev1.SecretProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "hubble-server-certs",
												},
												Optional: &optional,
												Items: []corev1.KeyToPath{
													{
														Key:  "tls.crt",
														Path: "server.crt",
													},
													{
														Key:  "tls.key",
														Path: "server.key",
													},
												},
											},
											ConfigMap: &corev1.ConfigMapProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "hubble-ca-cert",
												},
												Optional: &optional,
												Items: []corev1.KeyToPath{
													{
														Key:  "ca.crt",
														Path: "client-ca.crt",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
