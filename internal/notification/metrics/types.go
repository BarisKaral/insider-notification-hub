package metrics

import "github.com/prometheus/client_golang/prometheus"

// NotificationMetrics holds Prometheus metrics for notification processing.
type NotificationMetrics struct {
	NotificationsTotal *prometheus.CounterVec
	ProcessingDuration *prometheus.HistogramVec
	QueueDepth         *prometheus.GaugeVec
}
