package diagnostics

import (
	"context"

	diag_utils "github.com/dapr/dapr/pkg/diagnostics/utils"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

type resiliencyMetrics struct {
	// Circuit Breaker
	cbOpenCount       *stats.Int64Measure
	cbTooManyReqCount *stats.Int64Measure

	appID     string
	ctx       context.Context
	enabled   bool
	namespace string
}

func newResiliencyMetrics() *resiliencyMetrics {
	return &resiliencyMetrics{
		cbOpenCount: stats.Int64(
			"resiliency/circuitbreaker_open/count",
			"The number of execution attempts in open state.",
			stats.UnitDimensionless),
		cbTooManyReqCount: stats.Int64(
			"resiliency/circuitbreaker_too_many_req/count",
			"The number of execution attempts in half-open state and request count is over maxRequest for cb",
			stats.UnitDimensionless),
	}
}

// Init registers the resiliency metrics views.
func (m *resiliencyMetrics) Init(id string, namespace string) error {
	m.enabled = true
	m.appID = id
	m.namespace = namespace
	return view.Register(
		diag_utils.NewMeasureView(m.cbOpenCount, []tag.Key{appIDKey, componentKey, namespaceKey}, view.Count()),
		diag_utils.NewMeasureView(m.cbTooManyReqCount, []tag.Key{appIDKey, componentKey, namespaceKey}, view.Count()),
	)
}

// CircuitBreakerOpen records metric when trying to execute with open circuit breaker.
func (m *resiliencyMetrics) CircuitBreakerOpen(ctx context.Context, component string) {
	if m.enabled {
		_ = stats.RecordWithTags(
			ctx,
			diag_utils.WithTags(appIDKey, m.appID, componentKey, component, namespaceKey, m.namespace),
			m.cbOpenCount.M(1),
		)
	}
}

// CircuitBreakerHalfOpenTooManyReq records metric when trying to execute with half-open circuit breaker after too many requests.
func (m *resiliencyMetrics) CircuitBreakerHalfOpenTooManyReq(ctx context.Context, component string) {
	if m.enabled {
		_ = stats.RecordWithTags(
			ctx,
			diag_utils.WithTags(appIDKey, m.appID, componentKey, component, namespaceKey, m.namespace),
			m.cbTooManyReqCount.M(1),
		)
	}
}
