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

	// Name of the target resource.
	// Default is empty, which means selecting all resources.
	// +optional
	Name string `json:"name,omitempty"`

	// A label query over a set of resources.
	// If name is not empty, labelSelector will be ignored.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// State resource state
type State struct {
	// +required
	Name string `json:"name"`
}

// Transition state transition
type Transition struct {
	Source []string `json:"source"`
	Target string   `json:"target"`
	Event  string   `json:"event"`
}

type Match struct {
	Selector ResourceSelector `json:"selector"`
	// The input will cause an error if it does not follow this form:
	//
	//	<selector-syntax>         ::= <requirement> | <requirement> "," <selector-syntax>
	//	<requirement>             ::= [!] KEY [ <set-based-restriction> | <exact-match-restriction> ]
	//	<set-based-restriction>   ::= "" | <inclusion-exclusion> <value-set>
	//	<inclusion-exclusion>     ::= <inclusion> | <exclusion>
	//	<exclusion>               ::= "notin"
	//	<inclusion>               ::= "in"
	//	<value-set>               ::= "(" <values> ")"
	//	<values>                  ::= VALUE | VALUE "," <values>
	//	<exact-match-restriction> ::= ["="|"=="|"!="] VALUE
	//
	// KEY is a sequence of one or more characters following [ DNS_SUBDOMAIN "/" ] DNS_LABEL. Max length is 63 characters.
	// VALUE is a sequence of zero or more characters "([A-Za-z0-9_-\.])". Max length is 63 characters.
	// Delimiter is white space: (' ', '\t')
	// Example of valid syntax:
	//
	//	"x in (foo,,baz),y,z notin ()"
	// gjson express can be used in KEY
	LabelSelector string `json:"labelSelector,omitempty"`
	FieldSelector string `json:"fieldSelector,omitempty"`
}

// Event resource event
type Event struct {
	// +required
	Name string `json:"name"`

	// +required
	Matches []Match `json:"matches"`
}

// ResourceStateTransitionSpec defines the desired state of ResourceStateTransition
type ResourceStateTransitionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Selector restricts resource types that this StateTransition config applies to.
	Selector ResourceSelector `json:"selector"`

	States []State `json:"states"`

	Transitions []Transition `json:"transitions"`

	Events []Event `json:"events"`
}

// ResourceStateTransitionStatus defines the observed state of ResourceStateTransition
type ResourceStateTransitionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
