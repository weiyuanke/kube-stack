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
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	podlimiter "kube-stack.me/webhooks/pods/plugins/podlimiter"
)

// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,path=/validate-v1-pod,mutating=false,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=vpod.kb.io

type PodValidate struct {
	Client    client.Client
	ClientSet kubernetes.Interface
	decoder   *admission.Decoder
}

var (
	validatePlugins []ValidatePlugin = []ValidatePlugin{
		&podlimiter.PodlimiterPlugin{},
	}
)

func (v *PodValidate) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := v.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// TODO(user): your logic here
	for _, f := range validatePlugins {
		allow, msg, err := f.Validate(ctx, pod, req, v.Client, v.ClientSet)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if !allow {
			return admission.Denied(msg)
		}
	}

	return admission.Allowed("")
}

// InjectDecoder injects the decoder.
func (v *PodValidate) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
