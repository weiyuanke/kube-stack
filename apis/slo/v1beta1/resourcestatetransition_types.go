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

// ResourceSelector the resources which will be selected.
type ResourceSelector struct {
	// APIVersion represents the API version of the target resources.
	// +required
	APIVersion string `json:"apiVersion"`

	// Kind represents the Kind of the target resources.
	// +required
	Kind string `json:"kind"`

	// Namespace of the target resource.
	// Default is empty, which means inherit from the parent object scope.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// A selector to restrict the list of returned objects by their labels.
	// Defaults to everything.
	// +optional
	LabelSelector string `json:"labelSelector,omitempty" protobuf:"bytes,1,opt,name=labelSelector"`
	// A selector to restrict the list of returned objects by their fields.
	// Defaults to everything.
	// +optional
	FieldSelector string `json:"fieldSelector,omitempty" protobuf:"bytes,2,opt,name=fieldSelector"`
}

// Transition state transition
type Transition struct {
	Source []string `json:"source"`
	Target string   `json:"target"`
	Event  string   `json:"event"`
	// +optional
	NoMetric bool `json:"noMetric,omitempty"`
}

// Event resource event
type Event struct {
	// +required
	Name string `json:"name"`
	// +required
	Requirements []Requirement `json:"requirements"`
}

type TimerConfig struct {
	EventName      string `json:"eventName"`
	TimerInSeconds int    `json:"timerInSeconds"`
}

// ResourceStateTransitionSpec defines the desired state of ResourceStateTransition
type ResourceStateTransitionSpec struct {
	// Selector restricts resource types that this StateTransition config applies to.
	Selector ResourceSelector `json:"selector"`

	Transitions []Transition `json:"transitions"`

	Events []Event `json:"events"`

	// +optional
	Timer *TimerConfig `json:"timer,omitempty"`
}

// ResourceStateTransitionStatus defines the observed state of ResourceStateTransition
type ResourceStateTransitionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// ResourceStateTransition is the Schema for the resourcestatetransitions API
type ResourceStateTransition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceStateTransitionSpec   `json:"spec,omitempty"`
	Status ResourceStateTransitionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ResourceStateTransitionList contains a list of ResourceStateTransition
type ResourceStateTransitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceStateTransition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceStateTransition{}, &ResourceStateTransitionList{})
}
