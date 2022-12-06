package podlimiter

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	podlimiterv1 "kube-stack.me/apis/podlimiter/v1"
	"kube-stack.me/controllers/podlimiter"
)

type PodlimiterPlugin struct {
}

func (p *PodlimiterPlugin) Name() string {
	return "Podlimiter"
}

func (p *PodlimiterPlugin) Validate(ctx context.Context, obj *corev1.Pod, req admission.Request, c client.Client, clientSet kubernetes.Interface) (bool, string, error) {
	var limiters podlimiterv1.PodlimiterList
	if err := c.List(ctx, &limiters); err != nil {
		return false, "list podlimiter err", err
	}

	for _, limiter := range limiters.Items {
		for _, rule := range limiter.Spec.Rules {
			indexName := podlimiter.IndexName(&limiter, &rule)
			var pods corev1.PodList
			err := c.List(ctx, &pods, client.MatchingFields{indexName: podlimiter.Match})
			if err != nil || len(pods.Items) > rule.Threshhold {
				reason := fmt.Sprintf(
					"Over ThreashHold limiter %s rule %s threashhold %d current %d",
					limiter.Name, rule.Name, rule.Threshhold, len(pods.Items))
				return false, reason, nil
			}
		}
	}

	return true, "", nil
}
