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

package core

import (
	"context"
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"kube-stack.me/pkg/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	TABLE_EVENTUID_EVENTYAML = "eventuid-eventyaml"
	TABLE_PODUID_EVENTUIDS   = "poduid-eventuids"
	TABLE_NODENAME_EVENTUIDS = "nodename-eventuids"
)

var (
	eventlog = ctrl.Log.WithName("EventReconciler")
)

// EventReconciler reconciles a Event object
type EventReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=events/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Event object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *EventReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var event corev1.Event
	if err := r.Get(ctx, req.NamespacedName, &event); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// event uid -> event yaml
	if data, err := json.Marshal(event); err == nil {
		utils.Set(TABLE_EVENTUID_EVENTYAML, string(event.GetUID()), string(data))
	}

	// pod uid -> [event uid, event uid, ...]
	if event.InvolvedObject.GetObjectKind().GroupVersionKind().Kind == "Pod" {
		eventUIDSet := make(map[string]struct{})
		if data, err := utils.Get(TABLE_PODUID_EVENTUIDS, string(event.InvolvedObject.UID)); err == nil {
			json.Unmarshal([]byte(data), &eventUIDSet)
		}

		eventUIDSet[string(event.UID)] = struct{}{}
		data, _ := json.Marshal(eventUIDSet)
		utils.Set(TABLE_PODUID_EVENTUIDS, string(event.InvolvedObject.UID), string(data))
	}

	// node name -> [event uid, event uid, ...]
	if event.InvolvedObject.GetObjectKind().GroupVersionKind().Kind == "Node" {
		eventUIDSet := make(map[string]struct{})
		if data, err := utils.Get(TABLE_NODENAME_EVENTUIDS, string(event.InvolvedObject.Name)); err == nil {
			json.Unmarshal([]byte(data), &eventUIDSet)
		}

		eventUIDSet[string(event.UID)] = struct{}{}
		data, _ := json.Marshal(eventUIDSet)
		utils.Set(TABLE_NODENAME_EVENTUIDS, string(event.InvolvedObject.Name), string(data))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(r)
}
