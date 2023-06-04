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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"kube-stack.me/pkg/pod"
	"kube-stack.me/pkg/utils"
)

var (
	llog                    logr.Logger = ctrl.Log.WithName("PodReconciler")
	podsMap                 cache.ThreadSafeStore
	namespacedNameIndexName = "NamespacedNameIndexName"
)

func init() {
	indexers := cache.Indexers{
		namespacedNameIndexName: func(obj interface{}) ([]string, error) {
			return []string{obj.(*pod.PodState).NamespacedName.String()}, nil
		},
	}
	podsMap = cache.NewThreadSafeStore(indexers, cache.Indices{})

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for _, v := range podsMap.List() {
					if v.(*pod.PodState).Stopped {
						podsMap.Delete(string(v.(*pod.PodState).Pod.UID))
					}
				}
				podsMapSize.Set(float64(len(podsMap.ListKeys())))
			}
		}
	}()
}

// PodSLOReconciler reconciles a Pod object
type PodSLOReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PodSLOReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var po *corev1.Pod = &corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, po); err != nil {
		if errors.IsNotFound(err) {
			po = nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if po != nil {
		v, exists := podsMap.Get(string(po.UID))
		if exists {
			v.(*pod.PodState).ProcessEvent(po)
		} else {
			ps := pod.NewPodState(req.NamespacedName)
			ps.StartDispatching()
			podsMap.Add(string(po.UID), ps)
			ps.ProcessEvent(po)
		}
	} else {
		pss, err := podsMap.ByIndex(namespacedNameIndexName, req.NamespacedName.String())
		if err != nil || len(pss) <= 0 {
			llog.Info("No Pod by Index", "indexValue", req.NamespacedName.String())
			return ctrl.Result{}, nil
		}

		if len(pss) == 1 {
			pss[0].(*pod.PodState).ProcessEvent(po)
		} else {
			utils.SortPodStates(pss)
			for i := range pss {
				if i == 0 {
					pss[0].(*pod.PodState).ProcessEvent(po)
				} else {
					pss[i].(*pod.PodState).StopDispatching()
					podsMap.Delete(string(pss[i].(*pod.PodState).Pod.UID))
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodSLOReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
