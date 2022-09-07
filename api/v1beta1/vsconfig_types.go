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

// VSConfigSpec defines the desired state of VSConfig
type VSConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Target Kubernetes service name to trap requests
	Host string `json:"host"`
	// service to route when receiving http header `HeaderName: HeaderValue`
	Service string `json:"service"`
	// http header name to check
	HeaderName string `json:"headerName"`
	// http header value to route to Service
	HeaderValue string `json:"headerValue"`
}

// VSConfigStatus defines the observed state of VSConfig
type VSConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VSConfig is the Schema for the vsconfigs API
type VSConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VSConfigSpec   `json:"spec,omitempty"`
	Status VSConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VSConfigList contains a list of VSConfig
type VSConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VSConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VSConfig{}, &VSConfigList{})
}
