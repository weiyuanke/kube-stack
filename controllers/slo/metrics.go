package slo

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	listAllDurMetrics = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "list_all_resource_dur",
		},
		[]string{"resource"},
	)
	watchEventCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "watch_event_recv_total",
		},
		[]string{"resource", "type"},
	)
	watchEventDelayMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "watch_event_delay_ms",
			Buckets: []float64{5, 10, 30, 60, 100, 300, 600, 1000, 3000, 6000, 10000, 30000, 60000, 600000},
			Help:    "watch event delay in ms",
		},
		[]string{"resource"},
	)
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
	metrics.Registry.MustRegister(listAllDurMetrics)
	metrics.Registry.MustRegister(watchEventCounter)
	metrics.Registry.MustRegister(watchEventDelayMetrics)
	metrics.Registry.MustRegister(resourceStateMapSize)
	metrics.Registry.MustRegister(currentStateNum)
	metrics.Registry.MustRegister(enterStateCounter)
}
