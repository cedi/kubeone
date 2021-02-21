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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func controllerDeployment(controllerImage string) []appsv1.Deployment {
	commonLabels := map[string]string{
		"io.cilium/app": "operator",
		"name":          "cilium-operator",
	}
	replicas := int32(1)
	maxUnavailable := intstr.FromInt(1)
	maxSurge := intstr.FromInt(1)
	valTrue := true

	return []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cilium-operator",
				Namespace: metav1.NamespaceSystem,
				Labels:    commonLabels,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: commonLabels,
				},
				Strategy: appsv1.DeploymentStrategy{
					Type: appsv1.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &appsv1.RollingUpdateDeployment{
						MaxUnavailable: &maxUnavailable,
						MaxSurge:       &maxSurge,
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: commonLabels,
					},
					Spec: v1.PodSpec{
						DeprecatedServiceAccount: "cilium-operator",
						ServiceAccountName:       "cilium-operator",
						PriorityClassName:        "system-cluster-critical",
						HostNetwork:              true,
						RestartPolicy:            v1.RestartPolicyAlways,
						Affinity: &v1.Affinity{
							PodAntiAffinity: &v1.PodAntiAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
									{
										TopologyKey: v1.LabelHostname,
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      "io.cilium/app",
													Operator: metav1.LabelSelectorOpIn,
													Values:   []string{"operator"},
												},
											},
										},
									},
								},
							},
						},
						Tolerations: []v1.Toleration{
							{
								Operator: v1.TolerationOpExists,
							},
						},
						Volumes: []v1.Volume{
							{
								Name: "cilium-config-path",
								VolumeSource: v1.VolumeSource{
									ConfigMap: &v1.ConfigMapVolumeSource{
										LocalObjectReference: v1.LocalObjectReference{
											Name: "cilium-config",
										},
									},
								},
							},
						},
						Containers: []v1.Container{
							{
								Name:            "cilium-operator",
								ImagePullPolicy: v1.PullAlways,
								Image:           controllerImage,
								Command:         []string{"cilium-operator-generic"},
								Args: []string{
									"--config-dir=/tmp/cilium/config-map",
									"--debug=$(CILIUM_DEBUG)",
								},
								Env: []v1.EnvVar{
									{
										Name: "K8S_NODE_NAME",
										ValueFrom: &v1.EnvVarSource{
											FieldRef: &v1.ObjectFieldSelector{
												APIVersion: "v1",
												FieldPath:  "spec.nodeName",
											},
										},
									},
									{
										Name: "CILIUM_K8S_NAMESPACE",
										ValueFrom: &v1.EnvVarSource{
											FieldRef: &v1.ObjectFieldSelector{
												APIVersion: "v1",
												FieldPath:  "metadata.namespace",
											},
										},
									},
									{
										Name: "CILIUM_DEBUG",
										ValueFrom: &v1.EnvVarSource{
											ConfigMapKeyRef: &v1.ConfigMapKeySelector{
												Key:      "debug",
												Optional: &valTrue,
												LocalObjectReference: v1.LocalObjectReference{
													Name: "cilium-config",
												},
											},
										},
									},
								},
								LivenessProbe: &v1.Probe{
									InitialDelaySeconds: 60,
									PeriodSeconds:       10,
									TimeoutSeconds:      3,
									Handler: v1.Handler{
										HTTPGet: &v1.HTTPGetAction{
											Path: "/healthz",
											Port: intstr.IntOrString{
												IntVal: 9234,
												Type:   intstr.Int,
											},
											Host:   "127.0.0.1",
											Scheme: v1.URISchemeHTTP,
										},
									},
								},
								VolumeMounts: []v1.VolumeMount{
									{
										Name:      "cilium-config-path",
										ReadOnly:  true,
										MountPath: "/tmp/cilium/config-map",
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
