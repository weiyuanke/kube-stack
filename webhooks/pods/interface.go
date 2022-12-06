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

package pods

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// MutatePlugin pod mutate plugin
type MutatePlugin interface {
	Name() string
	Admission(ctx context.Context, obj *corev1.Pod, req admission.Request, client client.Client, clientSet kubernetes.Interface) error
}

var _ MutatePlugin = &MutatePluginFuncs{}

type MutatePluginFuncs struct {
	NameFunc      func() string
	AdmissionFunc func(ctx context.Context, obj *corev1.Pod, req admission.Request, client client.Client, clientSet kubernetes.Interface) error
}

func (p *MutatePluginFuncs) Name() string {
	return p.NameFunc()
}

func (p *MutatePluginFuncs) Admission(ctx context.Context, obj *corev1.Pod, req admission.Request, client client.Client, clientSet kubernetes.Interface) error {
	return p.AdmissionFunc(ctx, obj, req, client, clientSet)
}

// ValidatePlugin pod validate plugin
type ValidatePlugin interface {
	Name() string
	Validate(ctx context.Context, obj *corev1.Pod, req admission.Request, client client.Client, clientSet kubernetes.Interface) (bool, string, error)
}
