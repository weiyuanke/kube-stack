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
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubernetes/pkg/kubelet/prober"
	"k8s.io/kubernetes/pkg/kubelet/prober/results"
	status "k8s.io/kubernetes/pkg/kubelet/status"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	centralprobev1 "kube-stack.me/apis/centralprobe/v1"
)

// CentralProbeReconciler reconciles a CentralProbe object
type CentralProbeReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

var (
	llog             logr.Logger = ctrl.Log.WithName("CentralProbeReconciler")
	livenessManager  results.Manager
	readinessManager results.Manager
	startupManager   results.Manager
	proberManager    prober.Manager
)

const (
	finalizerName   string = "centralprobe.kube-stack.me/finalizer"
	podUIDIndexName string = "poduidindexname"
)

//+kubebuilder:rbac:groups=centralprobe.kube-stack.me,resources=centralprobes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=centralprobe.kube-stack.me,resources=centralprobes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=centralprobe.kube-stack.me,resources=centralprobes/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CentralProbe object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *CentralProbeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	var centralProbeList centralprobev1.CentralProbeList
	if err := r.List(ctx, &centralProbeList, client.InNamespace(req.Namespace)); err != nil {
		llog.Error(err, "failed to list centralProbe", "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	if len(centralProbeList.Items) > 0 {
		llog.Info("Reconcile", "namespace", req.Namespace, "number of CentralProbe", len(centralProbeList.Items))
	}

	hasError := false
	for _, cp := range centralProbeList.Items {
		if err := r.processCentralProbe(ctx, &cp); err != nil {
			llog.Error(err, "processCentralProbe", "centralProbe", cp)
			hasError = true
		}
	}

	if hasError {
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	return ctrl.Result{}, nil
}

func (r *CentralProbeReconciler) isNodeReady(node *corev1.Node) bool {
	for _, v := range node.Status.Conditions {
		if v.Type == corev1.NodeReady && v.Status == corev1.ConditionTrue && time.Since(v.LastHeartbeatTime.Time).Seconds() < 180 {
			return true
		}
	}
	return false
}

func (r *CentralProbeReconciler) updatePodStatus() {
	for result := range readinessManager.Updates() {
		llog.Info("Pod Status Changed", "probe result", result)

		var pods corev1.PodList
		err := r.List(context.TODO(), &pods, client.MatchingFields{podUIDIndexName: string(result.PodUID)})
		if err != nil || len(pods.Items) != 1 {
			llog.Error(err, "updatePodStatus error")
			continue
		}

		var node corev1.Node
		if err := r.Get(context.TODO(), types.NamespacedName{Name: pods.Items[0].Spec.NodeName}, &node); err != nil {
			llog.Error(err, "get Node err")
			continue
		}

		if r.isNodeReady(&node) {
			llog.Info("Node is Ready, Skip", "podName", pods.Items[0].Name, "nodeName", node.Name)
			continue
		}

		llog.Info("Node notReady, take over by centralProbe", "podName", pods.Items[0].Name, "nodeName", node.Name)

		podStatus := pods.Items[0].Status
		proberManager.UpdatePodStatus(result.PodUID, &podStatus)

		// updateConditionFunc updates the corresponding type of condition
		updateConditionFunc := func(conditionType corev1.PodConditionType, condition corev1.PodCondition) {
			conditionIndex := -1
			for i, condition := range podStatus.Conditions {
				if condition.Type == conditionType {
					conditionIndex = i
					break
				}
			}
			if conditionIndex != -1 {
				podStatus.Conditions[conditionIndex] = condition
			} else {
				llog.Info("PodStatus missing condition type", "conditionType", conditionType, "status", podStatus)
				podStatus.Conditions = append(podStatus.Conditions, condition)
			}
		}
		updateConditionFunc(corev1.PodReady, status.GeneratePodReadyCondition(&pods.Items[0].Spec, podStatus.Conditions, podStatus.ContainerStatuses, podStatus.Phase))
		updateConditionFunc(corev1.ContainersReady, status.GenerateContainersReadyCondition(&pods.Items[0].Spec, podStatus.ContainerStatuses, podStatus.Phase))

		pods.Items[0].Status = podStatus
		if err := r.Status().Update(context.TODO(), &pods.Items[0]); err != nil {
			llog.Error(err, "update Pod error")
		}
	}
}

func (r *CentralProbeReconciler) processCentralProbe(ctx context.Context, cp *centralprobev1.CentralProbe) error {
	var pods corev1.PodList
	if err := r.List(ctx, &pods, client.InNamespace(cp.Namespace), client.MatchingLabels(cp.Spec.Selector.MatchLabels)); err != nil {
		llog.Error(err, "unable to list pods")
		return err
	}

	// delete centralProbe
	if !cp.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(cp, finalizerName) {
			// cleanup prober workers
			for key := range cp.Status.ProbeStatuses {
				llog.Info("delete centralProbe, cleanup pods", "podName", key)
				proberManager.RemovePod(centralprobev1.ProbeStatustoPod(cp.Status.ProbeStatuses[key]))
			}
			controllerutil.RemoveFinalizer(cp, finalizerName)
			if err := r.Update(ctx, cp); err != nil {
				return err
			}
		}
		return nil
	}

	if !controllerutil.ContainsFinalizer(cp, finalizerName) {
		controllerutil.AddFinalizer(cp, finalizerName)
		if err := r.Update(ctx, cp); err != nil {
			return err
		}
	}

	if cp.Status.ProbeStatuses == nil {
		cp.Status.ProbeStatuses = make(map[string]*centralprobev1.ProbeStatus)
	}

	changed := false
	podNameSet := sets.NewString()
	// add pods
	for _, pod := range pods.Items {
		podNameSet.Insert(pod.Name)
		proberManager.AddPod(&pod)
		llog.Info("add pod to proberManager", "podName", pod.Name)
		if _, ok := cp.Status.ProbeStatuses[pod.Name]; !ok {
			changed = true
			cp.Status.ProbeStatuses[pod.Name] = centralprobev1.PodtoProbeStatus(&pod)
		}
	}
	// cleanup pods
	for name := range cp.Status.ProbeStatuses {
		if !podNameSet.Has(name) {
			changed = true
			proberManager.RemovePod(centralprobev1.ProbeStatustoPod(cp.Status.ProbeStatuses[name]))
			delete(cp.Status.ProbeStatuses, name)
			llog.Info("ProbeStatuses, remove pod...", "podName", name)
		}
	}
	if changed {
		if err := r.Status().Update(ctx, cp); err != nil {
			llog.Error(err, "update centralprobe status failed")
			return err
		}
	}
	return nil
}

func (r *CentralProbeReconciler) genQueueRequest(pod client.Object) []reconcile.Request {
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Namespace: pod.GetNamespace(),
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *CentralProbeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, podUIDIndexName, func(o client.Object) []string {
		return []string{string(o.GetUID())}
	}); err != nil {
		return err
	}

	// init prober manager
	livenessManager = results.NewManager()
	readinessManager = results.NewManager()
	startupManager = results.NewManager()
	proberManager = prober.NewManager(r, livenessManager, readinessManager, startupManager, nil, r.Recorder)
	go r.updatePodStatus()

	return ctrl.NewControllerManagedBy(mgr).
		For(&centralprobev1.CentralProbe{}).
		Watches(
			&source.Kind{Type: &corev1.Pod{}},
			handler.EnqueueRequestsFromMapFunc(r.genQueueRequest),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).Complete(r)
}
