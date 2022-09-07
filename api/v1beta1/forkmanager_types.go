/*
Copyright 2022.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Upstream struct {
	Host string `json:"host"`

	// Original server host
	// If empty, it will be assumed to be same af `Host`
	Original string `json:"original,omitempty"`

	// HostRewrite its value will rewrite `Host`
	HostRewrite string `json:"host_rewrite,omitempty"`
}

// ForkManagerSpec defines the desired state of ForkManager
type ForkManagerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// AmbassadorID to add Mappings
	AmbassadorID string `json:"ambassadorID"`

	// key of a HTTP header whose values is fork identifier
	// e.g. When headerKey = "X-Fork-Identifier" and the id is "some-id", Ambassador will add `X-Fork-Identifier: some-id` when accessed with `some-id` subdomain
	HeaderKey string `json:"headerKey"`

	// requests with header `Host: <fork-identifier>.<upstream-host>` will be propagated to `<upstream-host>`
	Upstreams []Upstream `json:"upstreams,omitempty"`
}

// ForkManagerStatus defines the observed state of ForkManager
type ForkManagerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ForkManager is the Schema for the forkmanagers API
type ForkManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ForkManagerSpec   `json:"spec,omitempty"`
	Status ForkManagerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ForkManagerList contains a list of ForkManager
type ForkManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ForkManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ForkManager{}, &ForkManagerList{})
}
