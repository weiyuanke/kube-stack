package podlimiter

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type PodlimiterPlugin struct {
}

func (p *PodlimiterPlugin) Name() string {
	return "Podlimiter"
}

func (p *PodlimiterPlugin) Validate(ctx context.Context, obj *corev1.Pod, req admission.Request, client client.Client, clientSet kubernetes.Interface) (bool, string, error) {
	return true, "", nil
}
