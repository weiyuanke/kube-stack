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

package centralprobe

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *CentralProbeReconciler) GetPodStatus(uid types.UID) (corev1.PodStatus, bool) {
	var pods corev1.PodList
	if err := r.List(context.TODO(), &pods, client.MatchingFields{podUIDIndexName: string(uid)}); err != nil {
		return corev1.PodStatus{}, false
	}
	if len(pods.Items) != 1 {
		return corev1.PodStatus{}, false
	}
	return pods.Items[0].Status, true
}
func (r *CentralProbeReconciler) Start() {}

func (r *CentralProbeReconciler) SetPodStatus(pod *corev1.Pod, status corev1.PodStatus) {}

func (r *CentralProbeReconciler) SetContainerReadiness(podUID types.UID, containerID kubecontainer.ContainerID, ready bool) {
}

func (r *CentralProbeReconciler) SetContainerStartup(podUID types.UID, containerID kubecontainer.ContainerID, started bool) {
}

func (r *CentralProbeReconciler) TerminatePod(pod *corev1.Pod) {}

func (r *CentralProbeReconciler) RemoveOrphanedStatuses(podUIDs map[types.UID]bool) {}
