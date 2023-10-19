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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	slov1beta1 "kube-stack.me/apis/slo/v1beta1"
)

const (
	labelKey = "apiserverslo.kube-stack.me/patchtimetsmil"
)

var (
	sloFinalizer     = slov1beta1.GroupVersion.Group + "/watchslo"
	sloReconcilerMap = cache.NewThreadSafeStore(cache.Indexers{}, cache.Indices{})
	delayTrackerMap  = cache.NewThreadSafeStore(cache.Indexers{}, cache.Indices{})
)

// WatchSLOReconciler reconciles a WatchSLO object
type WatchSLOReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	DynamicClient dynamic.Interface
}

//+kubebuilder:rbac:groups=slo.kube-stack.me,resources=watchslos,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=slo.kube-stack.me,resources=watchslos/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=slo.kube-stack.me,resources=watchslos/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *WatchSLOReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var wslo slov1beta1.WatchSLO
	if err := r.Get(ctx, req.NamespacedName, &wslo); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(&wslo, sloFinalizer) {
		controllerutil.AddFinalizer(&wslo, sloFinalizer)
		if err := r.Update(ctx, &wslo); err != nil {
			if errors.IsConflict(err) {
				return ctrl.Result{RequeueAfter: time.Second * 10}, nil
			}
			return ctrl.Result{}, err
		}
	}

	// process deletion
	if !wslo.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&wslo, sloFinalizer) {
			for _, config := range wslo.Spec.Configs {
				objKey := objKey(&wslo, config)
				// clean reconciler
				reconciler, exists := sloReconcilerMap.Get(objKey)
				if exists {
					reconciler.(*Reconciler).Stop()
					sloReconcilerMap.Delete(objKey)
				}
				//clean delay tacker
				if tracker, exists := delayTrackerMap.Get(objKey); exists {
					tracker.(*watchDelayTracker).Stop()
					delayTrackerMap.Delete(objKey)
				}
			}
			controllerutil.RemoveFinalizer(&wslo, sloFinalizer)
			if err := r.Update(ctx, &wslo); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, nil
	}

	// process create/update
	for i := range wslo.Spec.Configs {
		objKey := objKey(&wslo, wslo.Spec.Configs[i])

		// watch delay tracker
		if _, exists := delayTrackerMap.Get(objKey); !exists && wslo.Spec.Configs[i].TargetResource != "" {
			newT, err := newWatchDelayTracker(ctx, r.Client, r.DynamicClient, &wslo.Spec.Configs[i])
			if err != nil {
				return ctrl.Result{}, err
			}
			newT.Start()
			delayTrackerMap.Add(objKey, newT)
		}

		// reconciler for resource
		_, exists := sloReconcilerMap.Get(objKey)
		if !exists {
			newR, err := NewReconciler(ctx, r.Client, r.DynamicClient, wslo.Spec.Configs[i].Selector, r.handler)
			if err != nil {
				return ctrl.Result{}, err
			}
			go newR.Start()
			sloReconcilerMap.Add(objKey, newR)
		}
	}

	return ctrl.Result{}, nil
}

func getGroupVersionResource(clt client.Client, selector *slov1beta1.ResourceSelector) (*schema.GroupVersionResource, error) {
	// group version kind resource
	gv, err := schema.ParseGroupVersion(selector.APIVersion)
	if err != nil {
		return nil, err
	}

	gk := schema.GroupKind{Group: gv.Group, Kind: selector.Kind}
	restMapping, err := clt.RESTMapper().RESTMapping(gk, gv.Version)
	if err != nil {
		return nil, err
	}

	return &restMapping.Resource, nil
}

func objKey(c *slov1beta1.WatchSLO, config slov1beta1.SLOConfig) string {
	return fmt.Sprintf("%s-%s-%s-%s", client.ObjectKeyFromObject(c).String(), config.Selector.APIVersion, config.Selector.Kind, config.Selector.Namespace)
}

func (r *WatchSLOReconciler) handler(recon *Reconciler, e event) error {
	watchEventCounter.WithLabelValues(recon.selector.Kind, string(e.op)).Inc()
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WatchSLOReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&slov1beta1.WatchSLO{}).
		Complete(r)
}
