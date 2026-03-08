package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testMetrics bundles the NotificationMetrics with a custom registry for isolated tests.
type testMetrics struct {
	*NotificationMetrics
	registry *prometheus.Registry
}

// newTestMetrics creates a NotificationMetrics instance with a custom registry to avoid
// conflicts with the global default registry across tests.
func newTestMetrics(t *testing.T) *testMetrics {
	t.Helper()

	reg := prometheus.NewRegistry()

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

	reg.MustRegister(m.NotificationsTotal)
	reg.MustRegister(m.ProcessingDuration)
	reg.MustRegister(m.QueueDepth)

	return &testMetrics{NotificationMetrics: m, registry: reg}
}

// gatherMetricFamily gathers a metric family by name from the test registry.
func (tm *testMetrics) gatherMetricFamily(t *testing.T, name string) *dto.MetricFamily {
	t.Helper()
	families, err := tm.registry.Gather()
	require.NoError(t, err)
	for _, f := range families {
		if f.GetName() == name {
			return f
		}
	}
	return nil
}

func TestNewNotificationMetrics(t *testing.T) {
	m := newTestMetrics(t)

	assert.NotNil(t, m.NotificationsTotal)
	assert.NotNil(t, m.ProcessingDuration)
	assert.NotNil(t, m.QueueDepth)
}

func TestNewNotificationMetrics_FromConstructor(t *testing.T) {
	// Create a clean default registry for this test
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg
	defer func() {
		// Restore defaults
		prometheus.DefaultRegisterer = prometheus.NewRegistry()
		prometheus.DefaultGatherer = prometheus.NewRegistry()
	}()

	m := NewNotificationMetrics()

	require.NotNil(t, m)
	require.NotNil(t, m.NotificationsTotal)
	require.NotNil(t, m.ProcessingDuration)
	require.NotNil(t, m.QueueDepth)

	// Verify the metrics work
	m.IncTotal("sms", "sent")
	m.ObserveDuration("email", 100*time.Millisecond)
	m.SetQueueDepth("push", 5)

	// Verify via registry
	families, err := reg.Gather()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(families), 3)
}

func TestNewNotificationMetrics_ReturnsValidStruct(t *testing.T) {
	// Use a custom registry to avoid conflicts with default.
	// We can't call NewNotificationMetrics directly as it MustRegisters with default.
	// Instead, verify the helper methods work with different label combinations.
	m := newTestMetrics(t)

	// Test multiple channels for IncTotal.
	channels := []string{"sms", "email", "push"}
	statuses := []string{"sent", "failed", "pending"}
	for _, ch := range channels {
		for _, st := range statuses {
			m.IncTotal(ch, st)
		}
	}

	// Verify counts.
	for _, ch := range channels {
		for _, st := range statuses {
			var metric dto.Metric
			counter, err := m.NotificationsTotal.GetMetricWithLabelValues(ch, st)
			require.NoError(t, err)
			require.NoError(t, counter.Write(&metric))
			assert.Equal(t, float64(1), metric.GetCounter().GetValue())
		}
	}

	// Test ObserveDuration for all channels.
	for _, ch := range channels {
		m.ObserveDuration(ch, 100*time.Millisecond)
	}

	// Verify histograms.
	for _, ch := range channels {
		family := m.gatherMetricFamily(t, "notifications_processing_duration_seconds")
		require.NotNil(t, family, "metric family not found for channel %s", ch)
	}

	// Test SetQueueDepth for all channels.
	for _, ch := range channels {
		m.SetQueueDepth(ch, 5)
	}

	// Verify gauges.
	for _, ch := range channels {
		var metric dto.Metric
		gauge, err := m.QueueDepth.GetMetricWithLabelValues(ch)
		require.NoError(t, err)
		require.NoError(t, gauge.Write(&metric))
		assert.Equal(t, float64(5), metric.GetGauge().GetValue())
	}
}

func TestNotificationMetrics_IncTotal(t *testing.T) {
	m := newTestMetrics(t)

	m.IncTotal("email", "sent")
	m.IncTotal("email", "sent")
	m.IncTotal("sms", "failed")

	var metric dto.Metric

	emailSent, err := m.NotificationsTotal.GetMetricWithLabelValues("email", "sent")
	require.NoError(t, err)
	require.NoError(t, emailSent.Write(&metric))
	assert.Equal(t, float64(2), metric.GetCounter().GetValue())

	smsFailed, err := m.NotificationsTotal.GetMetricWithLabelValues("sms", "failed")
	require.NoError(t, err)
	metric.Reset()
	require.NoError(t, smsFailed.Write(&metric))
	assert.Equal(t, float64(1), metric.GetCounter().GetValue())
}

func TestNotificationMetrics_ObserveDuration(t *testing.T) {
	tm := newTestMetrics(t)

	tm.ObserveDuration("email", 150*time.Millisecond)
	tm.ObserveDuration("email", 250*time.Millisecond)

	family := tm.gatherMetricFamily(t, "notifications_processing_duration_seconds")
	require.NotNil(t, family, "metric family not found")
	require.Len(t, family.GetMetric(), 1)

	histogram := family.GetMetric()[0].GetHistogram()
	require.NotNil(t, histogram)
	assert.Equal(t, uint64(2), histogram.GetSampleCount())
	assert.InDelta(t, 0.4, histogram.GetSampleSum(), 0.01)
}

func TestNotificationMetrics_SetQueueDepth(t *testing.T) {
	m := newTestMetrics(t)

	m.SetQueueDepth("push", 42)

	var metric dto.Metric

	gauge, err := m.QueueDepth.GetMetricWithLabelValues("push")
	require.NoError(t, err)
	require.NoError(t, gauge.Write(&metric))
	assert.Equal(t, float64(42), metric.GetGauge().GetValue())

	// Update the value and verify it changes.
	m.SetQueueDepth("push", 10)
	metric.Reset()
	require.NoError(t, gauge.Write(&metric))
	assert.Equal(t, float64(10), metric.GetGauge().GetValue())
}
