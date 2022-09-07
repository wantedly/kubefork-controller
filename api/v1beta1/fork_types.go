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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ForkSpec defines the desired state of Fork
type ForkSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Pointer to ForkManager
	Manager string `json:"manager"`
	// A unique string to identify forked cluster, must be subdomain safe
	Identifier string `json:"identifier"`

	// Deadline is the time when fork will be removed.
	Deadline *metav1.Time `json:"deadline,omitempty"`

	GatewayOptions *GatewayOptions `json:"gatewayOptions,omitempty"`

	// service selector to copy
	Services *ForkService `json:"services,omitempty"`
	// deployment selector to copy
	Deployments *ForkDeployment `json:"deployments,omitempty"`
}

type ForkService struct {
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

type ForkDeployment struct {
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
	Template *PodTemplateSpec      `json:"template,omitempty"`
	Replicas *int32                `json:"replicas,omitempty"`
}

// PodTemplateSpec describes the data a pod should have when created from a template
type PodTemplateSpec struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	*metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the pod.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec PodSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// PodSpec is a subset of v1.PodSpec
type PodSpec struct {
	// List of containers belonging to the pod.
	// Containers cannot currently be added or removed.
	// There must be at least one container in a Pod.
	// Cannot be updated.
	// +patchMergeKey=name
	// +patchStrategy=merge
	Containers []v1.Container `json:"containers"`

	// Specifies the hostname of the Pod
	// If not specified, the pod's hostname will be set to a system-defined value.
	// +optional
	Hostname string `json:"hostname,omitempty"`
}

type GatewayOptions struct {
	// AddRequestHeaders will add headers in ambassador layer
	AddRequestHeaders map[string]string `json:"addRequestHeaders,omitempty"`
	AllowUpgrade      []string          `json:"allowUpgrade,omitempty"`
}

// ForkStatus defines the observed state of Fork
type ForkStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Fork is the Schema for the forks API
type Fork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ForkSpec   `json:"spec,omitempty"`
	Status ForkStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ForkList contains a list of Fork
type ForkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Fork `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Fork{}, &ForkList{})
}
