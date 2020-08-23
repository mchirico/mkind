/*
Copyright 2019 The Kubernetes Authors.

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

package patch

import (
	"testing"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestKubeYAML(t *testing.T) {
	t.Parallel()
	type testCase struct {
		Name            string
		ToPatch         string
		Patches         []string
		PatchesJSON6902 []config.PatchJSON6902
		ExpectError     bool
		ExpectOutput    string
	}
	cases := []testCase{
		{
			Name:         "kubeadm config no patches",
			ToPatch:      normalKubeadmConfig,
			ExpectError:  false,
			ExpectOutput: normalKubeadmConfigKustomized,
		},
		{
			Name:        "kubeadm config bogus patches",
			ToPatch:     normalKubeadmConfig,
			Patches:     []string{"b o g u s"},
			ExpectError: true,
		},
		{
			Name:         "kubeadm config one merge-patch",
			ToPatch:      normalKubeadmConfig,
			Patches:      []string{trivialPatch},
			ExpectError:  false,
			ExpectOutput: normalKubeadmConfigTrivialPatched,
		},
		{
			Name:            "kubeadm config one merg-patch, one 6902 patch",
			ToPatch:         normalKubeadmConfig,
			Patches:         []string{trivialPatch},
			PatchesJSON6902: []config.PatchJSON6902{trivialPatch6902},
			ExpectError:     false,
			ExpectOutput:    normalKubeadmConfigTrivialPatchedAnd6902Patched,
		},
	}
	for _, tc := range cases {
		tc := tc // capture test case
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			out, err := KubeYAML(tc.ToPatch, tc.Patches, tc.PatchesJSON6902)
			assert.ExpectError(t, tc.ExpectError, err)
			if err == nil {
				assert.StringEqual(t, tc.ExpectOutput, out)
			}
		})
	}
}

const normalKubeadmConfig = `# config generated by kind
apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
metadata:
  name: config
kubernetesVersion: v1.15.3
clusterName: "kind"
controlPlaneEndpoint: "192.168.9.3:6443"
# on docker for mac we have to expose the api server via port forward,
# so we need to ensure the cert is valid for localhost so we can talk
# to the cluster after rewriting the kubeconfig to point to localhost
apiServer:
  certSANs: [localhost, "127.0.0.1"]
controllerManager:
  extraArgs:
    enable-hostpath-provisioner: "true"
    # configure ipv6 default addresses for IPv6 clusters
    
scheduler:
  extraArgs:
    # configure ipv6 default addresses for IPv6 clusters
    
networking:
  podSubnet: "10.244.0.0/16"
  serviceSubnet: "10.96.0.0/12"
---
apiVersion: kubeadm.k8s.io/v1beta2
kind: InitConfiguration
metadata:
  name: config
# we use a well know token for TLS bootstrap
bootstrapTokens:
- token: "abcdef.0123456789abcdef"
# we use a well know port for making the API server discoverable inside docker network. 
# from the host machine such port will be accessible via a random local port instead.
localAPIEndpoint:
  advertiseAddress: "192.168.9.6"
  bindPort: 6443
nodeRegistration:
  criSocket: "/run/containerd/containerd.sock"
  kubeletExtraArgs:
    fail-swap-on: "false"
    node-ip: "192.168.9.6"
---
# no-op entry that exists solely so it can be patched
apiVersion: kubeadm.k8s.io/v1beta2
kind: JoinConfiguration
metadata:
  name: config
controlPlane:
  localAPIEndpoint:
    advertiseAddress: "192.168.9.6"
    bindPort: 6443
nodeRegistration:
  criSocket: "/run/containerd/containerd.sock"
  kubeletExtraArgs:
    fail-swap-on: "false"
    node-ip: "192.168.9.6"
discovery:
  bootstrapToken:
    apiServerEndpoint: "192.168.9.3:6443"
    token: "abcdef.0123456789abcdef"
    unsafeSkipCAVerification: true
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
metadata:
  name: config
# configure ipv6 addresses in IPv6 mode
 
# disable disk resource management by default
# kubelet will see the host disk that the inner container runtime
# is ultimately backed by and attempt to recover disk space. we don't want that.
imageGCHighThresholdPercent: 100
evictionHard:
  nodefs.available: "0%"
  nodefs.inodesFree: "0%"
  imagefs.available: "0%"
---
# no-op entry that exists solely so it can be patched
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
metadata:
  name: config
---`

const normalKubeadmConfigKustomized = `apiServer:
  certSANs:
  - localhost
  - 127.0.0.1
apiVersion: kubeadm.k8s.io/v1beta2
clusterName: kind
controlPlaneEndpoint: 192.168.9.3:6443
controllerManager:
  extraArgs:
    enable-hostpath-provisioner: "true"
kind: ClusterConfiguration
kubernetesVersion: v1.15.3
metadata:
  name: config
networking:
  podSubnet: 10.244.0.0/16
  serviceSubnet: 10.96.0.0/12
scheduler:
  extraArgs: null
---
apiVersion: kubeadm.k8s.io/v1beta2
bootstrapTokens:
- token: abcdef.0123456789abcdef
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: 192.168.9.6
  bindPort: 6443
metadata:
  name: config
nodeRegistration:
  criSocket: /run/containerd/containerd.sock
  kubeletExtraArgs:
    fail-swap-on: "false"
    node-ip: 192.168.9.6
---
apiVersion: kubeadm.k8s.io/v1beta2
controlPlane:
  localAPIEndpoint:
    advertiseAddress: 192.168.9.6
    bindPort: 6443
discovery:
  bootstrapToken:
    apiServerEndpoint: 192.168.9.3:6443
    token: abcdef.0123456789abcdef
    unsafeSkipCAVerification: true
kind: JoinConfiguration
metadata:
  name: config
nodeRegistration:
  criSocket: /run/containerd/containerd.sock
  kubeletExtraArgs:
    fail-swap-on: "false"
    node-ip: 192.168.9.6
---
apiVersion: kubelet.config.k8s.io/v1beta1
evictionHard:
  imagefs.available: 0%
  nodefs.available: 0%
  nodefs.inodesFree: 0%
imageGCHighThresholdPercent: 100
kind: KubeletConfiguration
metadata:
  name: config
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
metadata:
  name: config
`

const trivialPatch = `
kind: ClusterConfiguration
apiVersion: kubeadm.k8s.io/v1beta2

scheduler:
  extraArgs:
   some-extra-arg: the-arg
`

const normalKubeadmConfigTrivialPatched = `apiServer:
  certSANs:
  - localhost
  - 127.0.0.1
apiVersion: kubeadm.k8s.io/v1beta2
clusterName: kind
controlPlaneEndpoint: 192.168.9.3:6443
controllerManager:
  extraArgs:
    enable-hostpath-provisioner: "true"
kind: ClusterConfiguration
kubernetesVersion: v1.15.3
metadata:
  name: config
networking:
  podSubnet: 10.244.0.0/16
  serviceSubnet: 10.96.0.0/12
scheduler:
  extraArgs:
    some-extra-arg: the-arg
---
apiVersion: kubeadm.k8s.io/v1beta2
bootstrapTokens:
- token: abcdef.0123456789abcdef
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: 192.168.9.6
  bindPort: 6443
metadata:
  name: config
nodeRegistration:
  criSocket: /run/containerd/containerd.sock
  kubeletExtraArgs:
    fail-swap-on: "false"
    node-ip: 192.168.9.6
---
apiVersion: kubeadm.k8s.io/v1beta2
controlPlane:
  localAPIEndpoint:
    advertiseAddress: 192.168.9.6
    bindPort: 6443
discovery:
  bootstrapToken:
    apiServerEndpoint: 192.168.9.3:6443
    token: abcdef.0123456789abcdef
    unsafeSkipCAVerification: true
kind: JoinConfiguration
metadata:
  name: config
nodeRegistration:
  criSocket: /run/containerd/containerd.sock
  kubeletExtraArgs:
    fail-swap-on: "false"
    node-ip: 192.168.9.6
---
apiVersion: kubelet.config.k8s.io/v1beta1
evictionHard:
  imagefs.available: 0%
  nodefs.available: 0%
  nodefs.inodesFree: 0%
imageGCHighThresholdPercent: 100
kind: KubeletConfiguration
metadata:
  name: config
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
metadata:
  name: config
`

var trivialPatch6902 = config.PatchJSON6902{
	Group:   "kubeadm.k8s.io",
	Version: "v1beta2",
	Kind:    "ClusterConfiguration",
	Patch: `
- op: add
  path: /apiServer/certSANs/-
  value: my-hostname`,
}

const normalKubeadmConfigTrivialPatchedAnd6902Patched = `apiServer:
  certSANs:
  - localhost
  - 127.0.0.1
  - my-hostname
apiVersion: kubeadm.k8s.io/v1beta2
clusterName: kind
controlPlaneEndpoint: 192.168.9.3:6443
controllerManager:
  extraArgs:
    enable-hostpath-provisioner: "true"
kind: ClusterConfiguration
kubernetesVersion: v1.15.3
metadata:
  name: config
networking:
  podSubnet: 10.244.0.0/16
  serviceSubnet: 10.96.0.0/12
scheduler:
  extraArgs:
    some-extra-arg: the-arg
---
apiVersion: kubeadm.k8s.io/v1beta2
bootstrapTokens:
- token: abcdef.0123456789abcdef
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: 192.168.9.6
  bindPort: 6443
metadata:
  name: config
nodeRegistration:
  criSocket: /run/containerd/containerd.sock
  kubeletExtraArgs:
    fail-swap-on: "false"
    node-ip: 192.168.9.6
---
apiVersion: kubeadm.k8s.io/v1beta2
controlPlane:
  localAPIEndpoint:
    advertiseAddress: 192.168.9.6
    bindPort: 6443
discovery:
  bootstrapToken:
    apiServerEndpoint: 192.168.9.3:6443
    token: abcdef.0123456789abcdef
    unsafeSkipCAVerification: true
kind: JoinConfiguration
metadata:
  name: config
nodeRegistration:
  criSocket: /run/containerd/containerd.sock
  kubeletExtraArgs:
    fail-swap-on: "false"
    node-ip: 192.168.9.6
---
apiVersion: kubelet.config.k8s.io/v1beta1
evictionHard:
  imagefs.available: 0%
  nodefs.available: 0%
  nodefs.inodesFree: 0%
imageGCHighThresholdPercent: 100
kind: KubeletConfiguration
metadata:
  name: config
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
metadata:
  name: config
`
