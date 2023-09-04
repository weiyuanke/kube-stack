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

package v1beta1

import (
	"testing"
)

func TestMatches(t *testing.T) {
	// https://github.com/tidwall/gjson
	tests := []struct {
		name        string
		requirement Requirement
		args        string
		want        bool
	}{
		{
			requirement: Requirement{
				Key:       "status.conditions.#(type==Ready).status",
				Operator:  "in",
				StrValues: []string{"True"},
			},
			args: `{"metadata":{"$setElementOrder/finalizers":["protection-delete.pod.sigma.ali/naming-registered","finalizer.k8s.alipay.com/zappinfo","pod.beta1.sigma.ali/cni-allocated","finalizers.k8s.alipay.com/pod-fqdn"],"annotations":{"meta.k8s.alipay.com/trace-context":"[{\"trace_id\":\"f86136c0fa89a2700000000000000000\",\"parent_id\":\"\",\"root_span_id\":\"1820e9263f6a097e\",\"delivery_type\":\"PodCreate\",\"status\":\"closed\",\"services\":[{\"component\":\"cloud-scheduler\",\"span_id\":\"a7927b46959a5bf8\"},{\"component\":\"default-scheduler\",\"span_id\":\"2b3328ebe0bc363f\"},{\"component\":\"cni-service\",\"span_id\":\"899c038e43a62d7a\"},{\"component\":\"kubelet\",\"span_id\":\"fe32968d3986bb2a\"},{\"component\":\"zappinfo-controller\",\"span_id\":\"7d5c2485ab79855b\"},{\"component\":\"naming-controller\",\"span_id\":\"947883856f273c5c\"}],\"start_at\":\"2023-09-04T13:50:51+08:00\",\"finish_at\":\"2023-09-04T13:51:03+08:00\",\"extra_info\":null}]"},"resourceVersion":"3309939723"},"status":{"$setElementOrder/conditions":[{"type":"IPAllocated"},{"type":"NamingRegistered"},{"type":"Initialized"},{"type":"Ready"},{"type":"ContainersReady"},{"type":"ContainerDiskPressure"},{"type":"PodScheduled"}],"conditions":[{"lastTransitionTime":"2023-09-04T05:51:03Z","message":null,"reason":null,"status":"True","type":"Ready"}]}}`,
			want: true,
		},
		{
			requirement: Requirement{
				Key:       "apiVersion",
				Operator:  "in",
				StrValues: []string{""},
			},
			args: `{"apiVersion":null,"kind":null,"metadata":null,"spec":null,"status":null}`,
			want: true,
		},
		{
			requirement: Requirement{
				Key:       "spec.nodeName",
				Operator:  "notin",
				StrValues: []string{""},
			},
			args: `{"spec":{"nodeName":"testnode"}}`,
			want: true,
		},
		{
			requirement: Requirement{
				Key:       "children.0",
				Operator:  "in",
				StrValues: []string{"Sara"},
			},
			args: `
{
  "name": {"first": "Tom", "last": "Anderson"},
  "age":37,
  "children": ["Sara","Alex","Jack"],
  "fav.movie": "Deer Hunter",
  "friends": [
    {"first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
    {"first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
    {"first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
  ]
}`,
			want: true,
		},
		{
			requirement: Requirement{
				Key:       "children.#",
				Operator:  "in",
				StrValues: []string{"3"},
			},
			args: `
{
  "name": {"first": "Tom", "last": "Anderson"},
  "age":37,
  "children": ["Sara","Alex","Jack"],
  "fav.movie": "Deer Hunter",
  "friends": [
    {"first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
    {"first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
    {"first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
  ]
}`,
			want: true,
		},
		{
			requirement: Requirement{
				Key:       "age",
				Operator:  "in",
				StrValues: []string{"37"},
			},
			args: `
{
  "name": {"first": "Tom", "last": "Anderson"},
  "age":37,
  "children": ["Sara","Alex","Jack"],
  "fav.movie": "Deer Hunter",
  "friends": [
    {"first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
    {"first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
    {"first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
  ]
}`,
			want: true,
		},
		{
			requirement: Requirement{
				Key:       "age",
				Operator:  "in",
				StrValues: []string{"47"},
			},
			args: `{"name":{"first":"Janet","last":"Prichard"},"age":47}`,
			want: true,
		},
		{
			requirement: Requirement{
				Key:       "age",
				Operator:  "notin",
				StrValues: []string{"48"},
			},
			args: `{"name":{"first":"Janet","last":"Prichard"},"age":47}`,
			want: true,
		},
		{
			requirement: Requirement{
				Key:       "name.last",
				Operator:  "in",
				StrValues: []string{"Prichard"},
			},
			args: `{"name":{"first":"Janet","last":"Prichard"},"age":47}`,
			want: true,
		},
		{
			requirement: Requirement{
				Key:       "name.wei",
				Operator:  "in",
				StrValues: []string{""},
			},
			args: `{"name":{"first":"Janet","last":"Prichard"},"age":47}`,
			want: true,
		},
		{
			requirement: Requirement{
				Key:       "name.first",
				Operator:  "notin",
				StrValues: []string{""},
			},
			args: `{"name":{"first":"Janet","last":"Prichard"},"age":47}`,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := tt.requirement.Matches(tt.args)
			if re != tt.want {
				t.Errorf("TestMatch, %v, want %v", re, tt.want)
			}
		})
	}
}
