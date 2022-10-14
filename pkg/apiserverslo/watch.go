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

package apiserverslo

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	clientgowatch "k8s.io/client-go/tools/watch"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	llog logr.Logger = ctrl.Log.WithName("watch.go")
	unconfirmedts = sets.NewString()
	targetPodName string
	targetNamespace string
)

const (
	labelKey = "apiserverslo.kube-stack.me/patchtimetsmil"
)

func BindFlags(fs *flag.FlagSet) {
	fs.StringVar(&targetPodName, "target-pod-name", "", "specify target pod name")
	fs.StringVar(&targetNamespace, "target-namespace", "", "specify target namespace")
}

func StartWatchSLO(config *rest.Config) {
	clientSet := kubernetes.NewForConfigOrDie(config)
	go checkEventDelay(clientSet)
	go updatePodPeriodically(clientSet)
	go cleanTimestamp()
}

func checkEventDelay(clientSet *kubernetes.Clientset) {
	backoffManager := wait.NewExponentialBackoffManager(800*time.Millisecond, 30*time.Second, 2*time.Minute, 2.0, 1.0, &clock.RealClock{})
	wait.BackoffUntil(func() {
		startTime := time.Now()
		pods, err := clientSet.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{})
		if err != nil {
			llog.Error(err, "list all pods error")
			return
		}

		listAllDurationMetrics.WithLabelValues("pods").Observe(float64(time.Since(startTime).Milliseconds()))
		llog.Info("list all pods", "time cost", time.Since(startTime))

		watchFunc := func(opt v1.ListOptions) (watch.Interface, error) {
			return clientSet.CoreV1().Pods("").Watch(context.Background(), opt)
		}
		rw, _ := clientgowatch.NewRetryWatcher(pods.ResourceVersion, &cache.ListWatch{WatchFunc: watchFunc})
		for event := range rw.ResultChan() {
			watchEventCounter.WithLabelValues("pods", string(event.Type)).Inc()
			if event.Type == watch.Error {
				llog.Info("watch error event", "event", event)
				break
			}
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				continue
			}
			if pod.Namespace == targetNamespace && pod.Name == targetPodName && unconfirmedts.Has(pod.Labels[labelKey]) {
				unconfirmedts.Delete(pod.Labels[labelKey])
				unconfirmedTsMetrics.Set(float64(unconfirmedts.Len()))
				tsMil, _ := strconv.ParseInt(pod.Labels[labelKey], 10, 64)
				watchEventDelayMetrics.WithLabelValues("pods").Observe(float64(time.Now().UnixMilli() - tsMil))
				llog.Info("Watch Delay", "ms", time.Now().UnixMilli() - tsMil, "# left", unconfirmedts.Len(), "rv", pod.ResourceVersion, "ts", pod.Labels[labelKey])
			}
		}
	}, backoffManager, true, make(<-chan struct{}))
}

func cleanTimestamp() {
	wait.Until(func() {
		for ts := range unconfirmedts {
			tsMil, err := strconv.ParseInt(ts, 10, 64)
			if err != nil {
				unconfirmedts.Delete(ts)
				continue
			}
			if time.Now().UnixMilli() - tsMil > time.Second.Milliseconds() * 3600 {
				unconfirmedts.Delete(ts)
			}
			unconfirmedTsMetrics.Set(float64(unconfirmedts.Len()))
		}
	}, time.Second * 60, make(<-chan struct{}))
}

func updatePodPeriodically(clientSet *kubernetes.Clientset) {
	wait.Until(func() {
		timeStamp := time.Now().UnixMilli()
		patchData := fmt.Sprintf(`{"metadata":{"labels":{"%s":"%d"}}}`, labelKey, timeStamp)
		_, err := clientSet.CoreV1().Pods(targetNamespace).Patch(context.TODO(), targetPodName, types.MergePatchType, []byte(patchData), v1.PatchOptions{})
		if err != nil {
			llog.Error(err, "patch pod err")
			return
		}
		unconfirmedts.Insert(fmt.Sprintf("%d", timeStamp))
		unconfirmedTsMetrics.Set(float64(unconfirmedts.Len()))
		llog.Info("updatePodPeriodically", "ns", targetNamespace, "name", targetPodName, "tsMil", timeStamp, "nack#", unconfirmedts.Len())
	}, time.Second * 10, make(<-chan struct{}))
}
