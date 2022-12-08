package slo

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	podsMapSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "pods_map_size",
		},
	)
)

func init() {
	metrics.Registry.MustRegister(podsMapSize)
}
