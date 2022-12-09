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

package podlimiter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	podlimiterv1 "kube-stack.me/apis/podlimiter/v1"
	"kube-stack.me/pkg/utils"
)

// PodlimiterReconciler reconciles a Podlimiter object
type PodlimiterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	llog        = ctrl.Log.WithName("PodlimiterReconciler")
	podLimiters []podlimiterv1.Podlimiter
)

const (
	Match    = "true"
	NotMatch = "false"
)

//+kubebuilder:rbac:groups=podlimiter.kube-stack.me,resources=podlimiters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=podlimiter.kube-stack.me,resources=podlimiters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=podlimiter.kube-stack.me,resources=podlimiters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Podlimiter object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *PodlimiterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	// var pods corev1.PodList
	// r.List(context.TODO(), &pods, client.MatchingFields{"podlimiter-sample-test": "true"})
	// llog.Info("", "len", len(pods.Items))

	return ctrl.Result{}, nil
}

// ref kubernetes/pkg/apis/core/v1/conversion.go
func ConvertToFieldsSet(pod *corev1.Pod) *fields.Set {
	return &fields.Set{
		"metadata.name":            pod.Name,
		"metadata.namespace":       pod.Namespace,
		"spec.nodeName":            pod.Spec.NodeName,
		"spec.restartPolicy":       string(pod.Spec.RestartPolicy),
		"spec.schedulerName":       pod.Spec.SchedulerName,
		"spec.serviceAccountName":  pod.Spec.ServiceAccountName,
		"status.phase":             string(pod.Status.Phase),
		"status.podIP":             pod.Status.PodIP,
		"status.nominatedNodeName": pod.Status.NominatedNodeName,
	}
}

func ExtractFromRule(rule podlimiterv1.LimitRule) (labelSelector labels.Selector, fieldSelector fields.Selector) {
	var err error
	labelSelector, err = labels.Parse(rule.LabelSelector)
	if err != nil {
		llog.Error(err, fmt.Sprintf("invalid selector %v", rule.LabelSelector))
		os.Exit(1)
	}
	fieldSelector, err = fields.ParseSelector(rule.FieldSelector)
	if err != nil {
		llog.Error(err, fmt.Sprintf("invalid selector %v", rule.FieldSelector))
	}
	if labelSelector == nil {
		labelSelector = labels.Everything()
	}
	if fieldSelector == nil {
		fieldSelector = fields.Everything()
	}
	return labelSelector, fieldSelector
}

func IndexName(pl *podlimiterv1.Podlimiter, rule *podlimiterv1.LimitRule) string {
	return fmt.Sprintf("%s-%s", pl.Name, rule.Name)
}

func addIndex(mgr ctrl.Manager) error {
	var limiters podlimiterv1.PodlimiterList

	config := mgr.GetConfig()
	config.APIPath = "apis"
	config.GroupVersion = &podlimiterv1.GroupVersion
	config.NegotiatedSerializer = scheme.Codecs
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		llog.Error(err, "rest.RESTClientFor error")
		return err
	}

	if err := restClient.Get().Resource("podlimiters").Do(context.TODO()).Into(&limiters); err != nil {
		llog.Error(err, "list Podlimiter error")
		return err
	}

	for i := range limiters.Items {
		podLimiters = append(podLimiters, limiters.Items[i])
		for _, rule := range limiters.Items[i].Spec.Rules {
			labelSelect, fieldSelector := ExtractFromRule(rule)
			indexerFunc := func(o client.Object) []string {
				pod := o.(*corev1.Pod)
				if labelSelect.Matches(labels.Set(o.GetLabels())) && fieldSelector.Matches(ConvertToFieldsSet(pod)) {
					return []string{Match}
				}
				return []string{NotMatch}
			}
			if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, IndexName(&limiters.Items[i], &rule), indexerFunc); err != nil {
				llog.Error(err, "IndexField error", "rule", rule, "limiter name", limiters.Items[i].Name)
				os.Exit(1)
			}
		}
	}

	return nil
}

func refreshLimiterRuleState(mgr ctrl.Manager) {
	if success := mgr.GetCache().WaitForCacheSync(context.TODO()); !success {
		llog.Error(errors.New("refreshLimiterRuleState could not sync cache"), "")
		os.Exit(1)
	}

	wait.Until(func() {
		var podlimiters podlimiterv1.PodlimiterList
		if err := mgr.GetClient().List(context.TODO(), &podlimiters); err != nil {
			llog.Error(err, "PodlimiterList err")
			return
		}

		for _, pl := range podlimiters.Items {
			for _, rule := range pl.Spec.Rules {
				var pods corev1.PodList
				indexName := IndexName(&pl, &rule)
				if err := mgr.GetClient().List(context.TODO(), &pods, client.MatchingFields{indexName: Match}); err == nil {
					podlimiterRuleCurrentNum.WithLabelValues(pl.Name, rule.Name).Set(float64(len(pods.Items)))
				}
			}
		}
	}, time.Second*10, make(<-chan struct{}))
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodlimiterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := addIndex(mgr); err != nil {
		llog.Error(err, "AddIndex err")
		return nil
	}

	controller, err := utils.NewNonLeaderController("pod_limiter_controller", mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: 2,
	})
	if err != nil {
		return err
	}
	mgr.Add(controller)
	if err := controller.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// start collect limiter rule status
	go refreshLimiterRuleState(mgr)

	return nil
}
