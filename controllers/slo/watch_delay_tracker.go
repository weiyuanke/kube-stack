package slo

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	clientgowatch "k8s.io/client-go/tools/watch"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	slov1beta1 "kube-stack.me/apis/slo/v1beta1"
)

type watchDelayTracker struct {
	ctx                  context.Context
	client               client.Client
	dynamicClient        dynamic.Interface
	config               *slov1beta1.SLOConfig
	stopCh               chan struct{}
	groupVersionResource *schema.GroupVersionResource
	targetObject         *unstructured.Unstructured
}

func newWatchDelayTracker(cx context.Context, c client.Client, d dynamic.Interface, conf *slov1beta1.SLOConfig) (*watchDelayTracker, error) {
	gvr, err := getGroupVersionResource(c, &conf.Selector)
	if err != nil {
		return nil, err
	}

	var obj unstructured.Unstructured
	if err := yamlutil.Unmarshal([]byte(conf.TargetResource), &obj); err != nil {
		return nil, err
	}

	if _, err := d.Resource(*gvr).Namespace(obj.GetNamespace()).Create(cx, &obj, v1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			log.FromContext(cx).Error(err, "apply target resource error")
			return nil, err
		}
	}

	result := &watchDelayTracker{
		ctx:                  cx,
		client:               c,
		dynamicClient:        d,
		config:               conf,
		stopCh:               make(chan struct{}),
		groupVersionResource: gvr,
		targetObject:         &obj,
	}
	return result, nil
}

func (w *watchDelayTracker) Start() {
	unconfirmedts := sets.NewString()
	// update periodically
	go wait.Until(func() {
		timeStamp := time.Now().UnixMilli()
		unconfirmedts.Insert(fmt.Sprintf("%d", timeStamp))
		patchData := fmt.Sprintf(`{"metadata":{"labels":{"%s":"%d"}}}`, labelKey, timeStamp)
		if _, err := w.dynamicClient.Resource(*w.groupVersionResource).Namespace(w.targetObject.GetNamespace()).Patch(context.TODO(), w.targetObject.GetName(), types.MergePatchType, []byte(patchData), v1.PatchOptions{}); err != nil {
			log.FromContext(w.ctx).Error(err, "update error")
			return
		}
		log.FromContext(w.ctx).Info("updatePodPeriodically", "ts", timeStamp)
	}, time.Second*10, w.stopCh)

	// check delay
	backoffManager := wait.NewExponentialBackoffManager(800*time.Millisecond, 30*time.Second, 2*time.Minute, 2.0, 1.0, &clock.RealClock{})
	go wait.BackoffUntil(func() {
		startTime := time.Now()
		unstructure, err := w.dynamicClient.Resource(*w.groupVersionResource).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			return
		}
		log.FromContext(w.ctx).Info("list all pods", "time cost", time.Since(startTime))

		watchFunc := func(opt v1.ListOptions) (watch.Interface, error) {
			return w.dynamicClient.Resource(*w.groupVersionResource).Watch(context.TODO(), opt)
		}
		rw, _ := clientgowatch.NewRetryWatcher(unstructure.GetResourceVersion(), &cache.ListWatch{WatchFunc: watchFunc})
		for event := range rw.ResultChan() {
			if event.Type == watch.Error {
				break
			}

			eventObj := event.Object.(*unstructured.Unstructured)
			if eventObj.GetNamespace() != w.targetObject.GetNamespace() || eventObj.GetName() != w.targetObject.GetName() {
				continue
			}

			ts := eventObj.GetLabels()[labelKey]
			if unconfirmedts.Has(ts) {
				unconfirmedts.Delete(ts)
				tsMil, _ := strconv.ParseInt(ts, 10, 64)
				watchEventDelayMetrics.WithLabelValues(w.config.Selector.Kind).Observe(float64(time.Now().UnixMilli() - tsMil))
				log.FromContext(w.ctx).Info("Watch Delay", "delay", time.Now().UnixMilli()-tsMil, "nack#", unconfirmedts.Len(), "rv", eventObj.GetResourceVersion(), "ts", ts)
			}
		}
	}, backoffManager, true, w.stopCh)
}

func (w *watchDelayTracker) Stop() {
	close(w.stopCh)
}
