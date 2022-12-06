package pod

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	enterStateCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fsm_enter_state_count",
		},
		[]string{"state", "namespace"},
	)
	currentStateNum = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pod_state_num",
		},
		[]string{"state", "namespace"},
	)
)

func init() {
	metrics.Registry.MustRegister(enterStateCounter, currentStateNum)
}
