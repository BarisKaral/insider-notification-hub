package notification

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// NotificationMetrics holds Prometheus metrics for notification processing.
type NotificationMetrics struct {
	NotificationsTotal *prometheus.CounterVec
	ProcessingDuration *prometheus.HistogramVec
	QueueDepth         *prometheus.GaugeVec
}

// NewNotificationMetrics creates and registers Prometheus metrics for notifications.
func NewNotificationMetrics() *NotificationMetrics {
	m := &NotificationMetrics{
		NotificationsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_total",
				Help: "Total number of notifications processed, partitioned by channel and status.",
			},
			[]string{"channel", "status"},
		),
		ProcessingDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "notifications_processing_duration_seconds",
				Help:    "Duration of notification processing in seconds, partitioned by channel.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"channel"},
		),
		QueueDepth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "queue_depth",
				Help: "Current depth of notification queues, partitioned by channel.",
			},
			[]string{"channel"},
		),
	}

	prometheus.MustRegister(m.NotificationsTotal)
	prometheus.MustRegister(m.ProcessingDuration)
	prometheus.MustRegister(m.QueueDepth)

	return m
}

// IncTotal increments the total notifications counter for the given channel and status.
func (m *NotificationMetrics) IncTotal(channel, status string) {
	m.NotificationsTotal.WithLabelValues(channel, status).Inc()
}

// ObserveDuration records the processing duration for a notification on the given channel.
func (m *NotificationMetrics) ObserveDuration(channel string, duration time.Duration) {
	m.ProcessingDuration.WithLabelValues(channel).Observe(duration.Seconds())
}

// SetQueueDepth sets the current queue depth for the given channel.
func (m *NotificationMetrics) SetQueueDepth(channel string, depth float64) {
	m.QueueDepth.WithLabelValues(channel).Set(depth)
}
