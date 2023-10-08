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
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	slov1beta1 "kube-stack.me/apis/slo/v1beta1"
)

const (
	addOp        = "ADD"
	deleteOp     = "DELETE"
	updateOp     = "Update"
	indexName    = "NamespacedNameIndex"
	workerNumber = 16
)

type resourceReconciler struct {
	ctx                     context.Context
	client                  client.Client
	stopCh                  chan struct{}
	restMapping             *meta.RESTMapping
	queue                   workqueue.RateLimitingInterface
	informer                cache.SharedIndexInformer
	resourceStateTransition *slov1beta1.ResourceStateTransition
	resourceMap             cache.ThreadSafeStore
}

func newResourceReconciler(ctx context.Context, clt client.Client, dynamic dynamic.Interface, config *slov1beta1.ResourceStateTransition) (*resourceReconciler, error) {
	// list options
	tweatListOptions := func(options *metav1.ListOptions) {
		if config.Spec.Selector.LabelSelector != "" {
			options.LabelSelector = config.Spec.Selector.LabelSelector
		}
		if config.Spec.Selector.FieldSelector != "" {
			options.FieldSelector = config.Spec.Selector.FieldSelector
		}
	}

	indexers := cache.Indexers{
		indexName: func(obj interface{}) ([]string, error) {
			return []string{obj.(*ResourceState).NamespacedName.String()}, nil
		},
	}

	resourceReconciler := &resourceReconciler{
		ctx:                     ctx,
		client:                  clt,
		resourceStateTransition: config,
		stopCh:                  make(chan struct{}),
		queue:                   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		resourceMap:             cache.NewThreadSafeStore(indexers, cache.Indices{}),
	}

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// cleanup stopped ResourceState
				for _, v := range resourceReconciler.resourceMap.List() {
					if v.(*ResourceState).Stopped {
						resourceReconciler.resourceMap.Delete(string(v.(*ResourceState).Resource.GetUID()))
					}
				}
				resourceStateMapSize.WithLabelValues(config.Name).Set(float64(len(resourceReconciler.resourceMap.ListKeys())))
			case <-resourceReconciler.stopCh:
				return
			}
		}
	}()

	// group version kind resource
	gv, err := schema.ParseGroupVersion(config.Spec.Selector.APIVersion)
	if err != nil {
		return nil, err
	}

	gk := schema.GroupKind{Group: gv.Group, Kind: config.Spec.Selector.Kind}
	restMapping, err := clt.RESTMapper().RESTMapping(gk, gv.Version)
	if err != nil {
		return nil, err
	}

	resourceReconciler.restMapping = restMapping

	// construct informer
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamic, 0, config.Spec.Selector.Namespace, tweatListOptions)
	resourceReconciler.informer = factory.ForResource(restMapping.Resource).Informer()
	resourceReconciler.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			resourceReconciler.Enqueue(addOp, obj)
		},
		UpdateFunc: func(_, newObj interface{}) {
			resourceReconciler.Enqueue(updateOp, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			resourceReconciler.Enqueue(deleteOp, obj)
		},
	})

	factory.Start(resourceReconciler.stopCh)
	factory.WaitForCacheSync(resourceReconciler.stopCh)

	return resourceReconciler, nil
}

func (r *resourceReconciler) Start() {
	defer runtime.HandleCrash()
	defer r.queue.ShutDown()

	log.FromContext(r.ctx).Info("Starting workers")
	for i := 0; i < workerNumber; i++ {
		go wait.Until(r.runWorker, 0, r.stopCh)
	}

	log.FromContext(r.ctx).Info("Workers started")
	<-r.stopCh
	log.FromContext(r.ctx).Info("Shutting down workers")
}

func (r *resourceReconciler) Stop() {
	close(r.stopCh)
}

func (r *resourceReconciler) runWorker() {
	for r.processNextWorkItem() {
	}
}

func (r *resourceReconciler) processNextWorkItem() bool {
	elem, shutDown := r.queue.Get()
	if shutDown {
		return false
	}

	defer r.queue.Done(elem)

	err := r.Reconcile(elem.(event))
	if err == nil {
		r.queue.Forget(elem)
		return true
	}

	runtime.HandleError(err)
	r.queue.AddRateLimited(elem)

	return true
}

type event struct {
	op     string
	object *unstructured.Unstructured
}

func (r *resourceReconciler) Reconcile(e event) error {
	namespacedName := types.NamespacedName{
		Namespace: e.object.GetNamespace(),
		Name:      e.object.GetName(),
	}

	if e.op == deleteOp {
		pss, err := r.resourceMap.ByIndex(indexName, namespacedName.String())
		if err != nil || len(pss) <= 0 {
			log.FromContext(r.ctx).Info("No Resource by Index", "indexValue", namespacedName.String())
		}
		if len(pss) == 1 {
			pss[0].(*ResourceState).EnqueueEvent(nil)
		} else {
			sort.Slice(pss, func(i, j int) bool {
				return pss[i].(*ResourceState).CreateTime.After(pss[j].(*ResourceState).CreateTime)
			})
			for i := range pss {
				if i == 0 {
					pss[0].(*ResourceState).EnqueueEvent(nil)
				} else {
					pss[i].(*ResourceState).StopDispatching()
					r.resourceMap.Delete(string(pss[i].(*ResourceState).Resource.GetUID()))
				}
			}
		}
	} else {
		v, exists := r.resourceMap.Get(string(e.object.GetUID()))
		if !exists {
			rs, err := NewResourceState(r.ctx, namespacedName, r.resourceStateTransition)
			if err != nil {
				return err
			}
			r.resourceMap.Add(string(e.object.GetUID()), rs)
			go rs.StartDispatching()
			go rs.StartTimer()
			v = rs
		}
		v.(*ResourceState).EnqueueEvent(e.object)
	}
	return nil
}

func (r *resourceReconciler) Enqueue(op string, obj interface{}) {
	if obj == nil {
		return
	}

	e := event{
		op:     op,
		object: obj.(*unstructured.Unstructured),
	}
	r.queue.Add(e)
}
