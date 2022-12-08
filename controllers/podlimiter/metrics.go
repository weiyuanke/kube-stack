package podlimiter

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	podlimiterRuleCurrentNum = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "podlimiter_rule_current_num",
		},
		[]string{"podlimiter", "rule"},
	)
)

func init() {
	metrics.Registry.MustRegister(podlimiterRuleCurrentNum)
}
