package slo

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	DefaultWorkerNumber = 5
)

type Reconciler struct {
	ctx           context.Context
	client        client.Client
	dynamicClient dynamic.Interface
	selector      slov1beta1.ResourceSelector
	handler       handlerFunc
	stopCh        chan struct{}
	queue         workqueue.RateLimitingInterface
	informer      cache.SharedIndexInformer
}

type handlerFunc func(r *Reconciler, e event) error

func NewReconciler(cx context.Context, c client.Client, d dynamic.Interface, s slov1beta1.ResourceSelector, f handlerFunc) (*Reconciler, error) {
	result := &Reconciler{
		ctx:           cx,
		client:        c,
		dynamicClient: d,
		selector:      s,
		handler:       f,
		stopCh:        make(chan struct{}),
		queue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}

	// list options
	tweatListOptions := func(options *metav1.ListOptions) {
		if result.selector.LabelSelector != "" {
			options.LabelSelector = result.selector.LabelSelector
		}
		if result.selector.FieldSelector != "" {
			options.FieldSelector = result.selector.FieldSelector
		}
	}

	// groupVersionResource
	gvr, err := getGroupVersionResource(result.client, &result.selector)
	if err != nil {
		return nil, err
	}

	// construct informer
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(result.dynamicClient, 0, result.selector.Namespace, tweatListOptions)
	result.informer = factory.ForResource(*gvr).Informer()
	result.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			result.enqueue(addOp, obj)
		},
		UpdateFunc: func(_, newObj interface{}) {
			result.enqueue(updateOp, newObj)
		},
		DeleteFunc: func(obj interface{}) {
			result.enqueue(deleteOp, obj)
		},
	})

	factory.Start(result.stopCh)
	factory.WaitForCacheSync(result.stopCh)

	return result, nil
}

func (r *Reconciler) Start() {
	defer runtime.HandleCrash()
	defer r.queue.ShutDown()

	log.FromContext(r.ctx).Info("Starting workers")
	for i := 0; i < DefaultWorkerNumber; i++ {
		go wait.Until(r.runWorker, 0, r.stopCh)
	}

	log.FromContext(r.ctx).Info("Workers started")
	<-r.stopCh
	log.FromContext(r.ctx).Info("Shutting down workers")
}

func (r *Reconciler) runWorker() {
	for r.processNextWorkItem() {
	}
}

func (r *Reconciler) processNextWorkItem() bool {
	elem, shutDown := r.queue.Get()
	if shutDown {
		return false
	}

	defer r.queue.Done(elem)

	err := r.handler(r, elem.(event))
	if err == nil {
		r.queue.Forget(elem)
		return true
	}

	runtime.HandleError(err)
	r.queue.AddRateLimited(elem)

	return true
}

func (r *Reconciler) Stop() {
	close(r.stopCh)
}

func (r *Reconciler) enqueue(op string, obj interface{}) {
	e := event{
		op:     op,
		object: obj.(*unstructured.Unstructured),
	}
	r.queue.Add(e)
}
