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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"

	slov1beta1 "kube-stack.me/apis/slo/v1beta1"
)

const (
	requeuePeriod = time.Second
	workerNumber  = 16
)

type resourceReconciler struct {
	ctx                     context.Context
	client                  client.Client
	stopCh                  chan struct{}
	restMapping             *meta.RESTMapping
	queue                   workqueue.RateLimitingInterface
	informer                cache.SharedIndexInformer
	resourceStateTransition *slov1beta1.ResourceStateTransition
}

func newResourceReconciler(ctx context.Context, client client.Client, dynamic dynamic.Interface, config *slov1beta1.ResourceStateTransition) (*resourceReconciler, error) {
	// list options
	tweatListOptions := func(options *metav1.ListOptions) {
		if config.Spec.Selector.LabelSelector != "" {
			options.LabelSelector = config.Spec.Selector.LabelSelector
		}
		if config.Spec.Selector.FieldSelector != "" {
			options.FieldSelector = config.Spec.Selector.FieldSelector
		}
	}

	resourceReconciler := &resourceReconciler{
		ctx:                     ctx,
		client:                  client,
		resourceStateTransition: config,
		stopCh:                  make(chan struct{}),
		queue:                   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	// group version kind resource
	gv, err := schema.ParseGroupVersion(config.Spec.Selector.APIVersion)
	if err != nil {
		return nil, err
	}

	gk := schema.GroupKind{Group: gv.Group, Kind: config.Spec.Selector.Kind}
	restMapping, err := client.RESTMapper().RESTMapping(gk, gv.Version)
	if err != nil {
		return nil, err
	}

	resourceReconciler.restMapping = restMapping

	// construct informer
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamic, 0, config.Spec.Selector.Namespace, tweatListOptions)
	resourceReconciler.informer = factory.ForResource(restMapping.Resource).Informer()
	resourceReconciler.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			resourceReconciler.Enqueue("ADD", obj)
		},
		UpdateFunc: func(_, newObj interface{}) {
			resourceReconciler.Enqueue("UPDATE", newObj)
		},
		DeleteFunc: func(obj interface{}) {
			resourceReconciler.Enqueue("DELETE", obj)
		},
	})

	factory.Start(resourceReconciler.stopCh)
	factory.WaitForCacheSync(resourceReconciler.stopCh)

	return resourceReconciler, nil
}

func (r *resourceReconciler) Start() {
	for i := 0; i < workerNumber; i++ {
		go wait.Until(r.worker, 0, r.stopCh)
	}
	// Ensure all goroutines are cleaned up when the stop channel closes
	go func() {
		<-r.stopCh
		r.queue.ShutDown()
	}()
}

func (r *resourceReconciler) Stop() {
	close(r.stopCh)
}

func (r *resourceReconciler) worker() {
	for r.processNextWorkItem() {
	}
}

func (r *resourceReconciler) processNextWorkItem() bool {
	key, quit := r.queue.Get()
	if quit {
		return false
	}
	defer r.queue.Done(key)

	err := r.Reconcile(key.(*objKey))
	r.handleErr(err, key)

	return true
}

func (r *resourceReconciler) Reconcile(key *objKey) error {
	unstructuredObj := key.object.(*unstructured.Unstructured)
	llog.Info("", "xxxx========", unstructuredObj)
	return nil
}

type objKey struct {
	event  string
	object runtime.Object
}

func (r *resourceReconciler) Enqueue(event string, obj interface{}) {
	if obj == nil {
		return
	}

	key := objKey{
		event:  event,
		object: obj.(runtime.Object),
	}
	r.queue.Add(&key)
}

func (r *resourceReconciler) handleErr(err error, key interface{}) {
	if err == nil {
		r.queue.Forget(key)
		return
	}

	r.queue.AddAfter(key, requeuePeriod)
}
