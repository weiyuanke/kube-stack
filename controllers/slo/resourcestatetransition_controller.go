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

package slo

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	slov1beta1 "kube-stack.me/apis/slo/v1beta1"
)

var (
	finalizer     = slov1beta1.GroupVersion.Group + "/resource-state-finalizer"
	reconcilerMap = cache.NewThreadSafeStore(cache.Indexers{}, cache.Indices{})
)

// ResourceStateTransitionReconciler reconciles a ResourceStateTransition object
type ResourceStateTransitionReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	DynamicClient dynamic.Interface
}

//+kubebuilder:rbac:groups=slo.kube-stack.me,resources=resourcestatetransitions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=slo.kube-stack.me,resources=resourcestatetransitions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=slo.kube-stack.me,resources=resourcestatetransitions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ResourceStateTransitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	defer func() {
		llog.Info("", "ResourceReconciler Number", len(reconcilerMap.ListKeys()))
	}()

	var resourceStateTransition slov1beta1.ResourceStateTransition
	if err := r.Get(ctx, req.NamespacedName, &resourceStateTransition); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(&resourceStateTransition, finalizer) {
		controllerutil.AddFinalizer(&resourceStateTransition, finalizer)
		if err := r.Update(ctx, &resourceStateTransition); err != nil {
			if errors.IsConflict(err) {
				return ctrl.Result{RequeueAfter: time.Second * 10}, nil
			}
			return ctrl.Result{}, err
		}
	}

	objKey := client.ObjectKeyFromObject(&resourceStateTransition).String()

	// process deletion
	if !resourceStateTransition.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&resourceStateTransition, finalizer) {
			reconciler, exists := reconcilerMap.Get(objKey)
			if exists {
				reconciler.(*resourceReconciler).Stop()
				reconcilerMap.Delete(objKey)
			}

			controllerutil.RemoveFinalizer(&resourceStateTransition, finalizer)
			if err := r.Update(ctx, &resourceStateTransition); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}

	// process create/update
	reconciler, exists := reconcilerMap.Get(objKey)
	if !exists {
		newReconciler, err := newResourceReconciler(ctx, r.Client, r.DynamicClient, &resourceStateTransition)
		if err != nil {
			return ctrl.Result{}, err
		}
		newReconciler.Start()
		reconcilerMap.Add(objKey, newReconciler)
		return ctrl.Result{}, nil
	}

	rv := reconciler.(*resourceReconciler).resourceStateTransition.ResourceVersion
	if resourceStateTransition.ResourceVersion == rv {
		return ctrl.Result{}, nil
	}

	newReconciler, err := newResourceReconciler(ctx, r.Client, r.DynamicClient, &resourceStateTransition)
	if err != nil {
		return ctrl.Result{}, err
	}
	newReconciler.Start()
	reconcilerMap.Delete(objKey)
	reconcilerMap.Add(objKey, newReconciler)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceStateTransitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&slov1beta1.ResourceStateTransition{}).
		Complete(r)
}
