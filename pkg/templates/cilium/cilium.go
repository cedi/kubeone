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
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"k8c.io/kubeone/pkg/clientutil"
	"k8c.io/kubeone/pkg/kubeconfig"
	"k8c.io/kubeone/pkg/state"
)

const (
	ciliumImageRegistry     = "quay.io"
	ciliumInstallImageName  = "cilium/cilium:latest"
	ciliumImageName         = "cilium/operator-generic:latest"
	ciliumoperatorImageName = "cilium/operator-generic:latest"
)

// Deploy deploys Cilium CNI on the cluster
func Deploy(s *state.State) error {
	if s.DynamicClient == nil {
		return errors.New("kubernetes dynamic client is not initialized")
	}

	ctx := context.Background()

	installCNIImage := s.Cluster.RegistryConfiguration.ImageRegistry(ciliumImageRegistry) + ciliumInstallImageName
	ciliumImage := s.Cluster.RegistryConfiguration.ImageRegistry(ciliumImageRegistry) + ciliumImageName
	controllerImage := s.Cluster.RegistryConfiguration.ImageRegistry(ciliumImageRegistry) + ciliumoperatorImageName

	k8sObjects := []runtime.Object{}

	k8sObjects = append(k8sObjects,
		// ServiceAccounts
		serviceAccountCilium(),
		serviceAccountCiliumOperator(),

		// ConfigMap
		configMap(),

		// ClusterRole
		clusterRoleCilium(),
		clusterRoleCiliumOperator(),

		// ClusterRoleBinding
		clusterRoleBindingCilium(),
		clusterRoleBindingCiliumOperator(),

		// DaemonSet
		daemonSet(installCNIImage, ciliumImage),

		// Deployment
		controllerDeployment(controllerImage),
	)

	for _, obj := range k8sObjects {
		if err := clientutil.CreateOrUpdate(ctx, s.DynamicClient, obj); err != nil {
			return errors.WithStack(err)
		}
	}

	// HACK: re-init dynamic client in order to re-init RestMapper, to drop caches
	err := kubeconfig.HackIssue321InitDynamicClient(s)
	return errors.Wrap(err, "failed to re-init dynamic client")
}
