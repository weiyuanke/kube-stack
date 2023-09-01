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
	"github.com/tidwall/gjson"
)

type Operator string

const (
	In    Operator = "in"
	NotIn Operator = "notin"
)

type Requirement struct {
	Key       string   `json:"key"`
	Operator  Operator `json:"operator"`
	StrValues []string `json:"strValues"`
}

func (r *Requirement) hasValue(value string) bool {
	for i := range r.StrValues {
		if r.StrValues[i] == value {
			return true
		}
	}
	return false
}

func (r *Requirement) Matches(jsonData string) bool {
	v := gjson.Get(jsonData, r.Key)
	str := v.String()
	switch r.Operator {
	case In:
		return r.hasValue(str)
	case NotIn:
		return !r.hasValue(str)
	default:
		return false
	}
}
