package cli

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	gauge           *prometheus.GaugeVec
	apiErrorCounter *prometheus.CounterVec
)

func BuildGauge() {
	gauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "alertsvalidator_validity_range",
			Help: "Does the alert's metric contains data point in range",
		},
		append([]string{
			"alertname",
			"alertid",
			"range_from",
			"range_to",
			"status",
		}, Conf.LabelKeys...),
	)
	apiErrorCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertsvalidator_external_api_error",
			Help: "Are there data fetching errors",
		},
		append([]string{
			"type",
			"server",
		}, Conf.LabelKeys...),
	)
}
