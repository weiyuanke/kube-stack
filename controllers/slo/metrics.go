package slo

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	resourceStateMapSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "resourcestate_map_size",
		},
		[]string{"transName"},
	)
	enterStateCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "resourcestate_total_num",
		},
		[]string{"transName", "state"},
	)
	currentStateNum = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "resourcestate_current_num",
		},
		[]string{"transName", "state"},
	)
)

func init() {
	metrics.Registry.MustRegister(resourceStateMapSize)
	metrics.Registry.MustRegister(currentStateNum)
	metrics.Registry.MustRegister(enterStateCounter)
}
