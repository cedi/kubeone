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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func clusterRole() []rbacv1.ClusterRole {
	clusterRoleCilium := clusterRoleCilium()
	clusterRoleCiliumOperator := clusterRoleCiliumOperator()

	return []rbacv1.ClusterRole{clusterRoleCilium, clusterRoleCiliumOperator}
}

func clusterRoleCilium() rbacv1.ClusterRole {
	defaultVerbs := []string{"get", "list", "watch"}

	return rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cilium",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"networking.k8s.io"},
				Resources: []string{"networkpolicies"},
				Verbs:     defaultVerbs,
			},
			{
				APIGroups: []string{"discovery.k8s.io"},
				Resources: []string{"endpointslices"},
				Verbs:     defaultVerbs,
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces", "services", "nodes", "endpoints"},
				Verbs:     defaultVerbs,
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "pods/finalizers"},
				Verbs:     defaultVerbs,
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "pods/finalizers"},
				Verbs:     []string{"get", "list", "watch", "update", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes", "nodes/status"},
				Verbs:     []string{"patch"},
			},
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"tomresourcedefinitions"},
				Verbs: []string{
					// Deprecated for removal in v1.10
					"create",
					"list",
					"watch",
					"update",

					// This is used when validating policies in preflight. This will need to stay
					// until we figure out how to avoid "get" inside the preflight, and then
					// should be removed ideally.
					"get",
				},
			},
			{
				APIGroups: []string{"cilium.io"},
				Resources: []string{
					"ciliumnetworkpolicies",
					"ciliumnetworkpolicies/status",
					"ciliumnetworkpolicies/finalizers",
					"ciliumclusterwidenetworkpolicies",
					"ciliumclusterwidenetworkpolicies/status",
					"ciliumclusterwidenetworkpolicies/finalizers",
					"ciliumendpoints",
					"ciliumendpoints/status",
					"ciliumendpoints/finalizers",
					"ciliumnodes",
					"ciliumnodes/status",
					"ciliumnodes/finalizers",
					"ciliumidentities",
					"ciliumidentities/finalizers",
					"ciliumlocalredirectpolicies",
					"ciliumlocalredirectpolicies/status",
					"ciliumlocalredirectpolicies/finalizers",
				},
				Verbs: []string{"*"},
			},
		},
	}
}

func clusterRoleCiliumOperator() rbacv1.ClusterRole {
	return rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cilium-operator",
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch", "delete"},
			},
			{
				APIGroups: []string{"discovery.k8s.io"},
				Resources: []string{"endpointslices"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"discovery.k8s.io"},
				Resources: []string{"endpointslices"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{
					// to perform the translation of a CNP that contains `ToGroup` to its endpoints
					"services",
					"endpoints",
					// to check apiserver connectivity
					"namespaces",
				},
				Verbs: []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"cilium.io"},
				Resources: []string{
					"ciliumnetworkpolicies",
					"ciliumnetworkpolicies/status",
					"ciliumnetworkpolicies/finalizers",
					"ciliumclusterwidenetworkpolicies",
					"ciliumclusterwidenetworkpolicies/status",
					"ciliumclusterwidenetworkpolicies/finalizers",
					"ciliumendpoints",
					"ciliumendpoints/status",
					"ciliumendpoints/finalizers",
					"ciliumnodes",
					"ciliumnodes/status",
					"ciliumnodes/finalizers",
					"ciliumidentities",
					"ciliumidentities/status",
					"ciliumidentities/finalizers",
					"ciliumlocalredirectpolicies",
					"ciliumlocalredirectpolicies/status",
					"ciliumlocalredirectpolicies/finalizers",
				},
				Verbs: []string{"*"},
			},
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"create", "get", "list", "update", "watch"},
			},
			// For cilium-operator running in HA mode.
			//
			// Cilium operator running in HA mode requires the use of ResourceLock for Leader Election
			// between mulitple running instances.
			// The preferred way of doing this is to use LeasesResourceLock as edits to Leases are less
			// common and fewer objects in the cluster watch "all Leases".
			// The support for leases was introduced in coordination.k8s.io/v1 during Kubernetes 1.14 release.
			// In Cilium we currently don't support HA mode for K8s version < 1.14. This condition make sure
			// that we only authorize access to leases resources in supported K8s versions.
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"create", "get", "update"},
			},
		},
	}
}
