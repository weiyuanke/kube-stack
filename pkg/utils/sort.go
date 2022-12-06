package utils

import (
	"sort"

	"kube-stack.me/pkg/pod"
)

func SortPodStates(pss []interface{}) {
	sort.Slice(pss, func(i, j int) bool {
		return pss[i].(*pod.PodState).CreateTime.After(pss[j].(*pod.PodState).CreateTime)
	})
}
