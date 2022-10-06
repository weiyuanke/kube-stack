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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CentralProbeSpec defines the desired state of CentralProbe
type CentralProbeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Selector *metav1.LabelSelector `json:"selector"`
}

// CentralProbeStatus defines the observed state of CentralProbe
type CentralProbeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ProbeStatuses map[string]*ProbeStatus `json:"probeStatuses"`
}

type ProbeStatus struct {
	Result         string   `json:"result"`
	PodUID         string   `json:"podUID"`
	PodName        string   `json:"podName"`
	ContainerNames []string `json:"containerNames,omitempty"`
}

const (
	Probing = "Probing"
)

func ProbeStatustoPod(in *ProbeStatus) *corev1.Pod {
	pod := &corev1.Pod{}
	pod.SetUID(types.UID(in.PodUID))
	pod.SetName(in.PodName)
	containers := make([]corev1.Container, 0)
	for _, n := range in.ContainerNames {
		containers = append(containers, corev1.Container{
			Name: n,
		})
	}
	pod.Spec.Containers = containers
	return pod
}

func PodtoProbeStatus(pod *corev1.Pod) *ProbeStatus {
	ps := &ProbeStatus{}
	ps.PodUID = string(pod.UID)
	ps.PodName = pod.Name
	names := make([]string, 0)
	for _, c := range pod.Spec.Containers {
		names = append(names, c.Name)
	}
	ps.ContainerNames = names
	return ps
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CentralProbe is the Schema for the centralprobes API
type CentralProbe struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CentralProbeSpec   `json:"spec,omitempty"`
	Status CentralProbeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CentralProbeList contains a list of CentralProbe
type CentralProbeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CentralProbe `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CentralProbe{}, &CentralProbeList{})
}
