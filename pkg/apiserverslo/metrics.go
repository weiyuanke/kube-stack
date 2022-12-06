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

package apiserverslo

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	unconfirmedTsMetrics = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "unconfirmed_timestamp_num",
			Help: "describe the metrics",
		},
	)
	watchEventDelayMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "watch_event_delay_ms",
			Buckets: []float64{5, 10, 30, 60, 100, 300, 600, 1000, 3000, 6000, 10000, 30000, 60000, 600000},
			Help:    "watch event delay in ms",
		},
		[]string{"resource"},
	)
	listAllDurationMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "list_all_resource_duration_ms",
			Buckets: []float64{5, 10, 30, 60, 100, 300, 600, 1000, 3000, 6000, 10000, 30000, 60000, 600000},
		},
		[]string{"resource"},
	)
	watchEventCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "watch_event_recv_total",
		},
		[]string{"resource", "type"},
	)
)

func init() {
	metrics.Registry.MustRegister(unconfirmedTsMetrics)
	metrics.Registry.MustRegister(watchEventDelayMetrics)
	metrics.Registry.MustRegister(listAllDurationMetrics)
	metrics.Registry.MustRegister(watchEventCounter)
}
