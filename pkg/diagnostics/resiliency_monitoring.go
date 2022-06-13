package diagnostics

import (
	"context"

	diag_utils "github.com/dapr/dapr/pkg/diagnostics/utils"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	CircuitBreakerPolicy PolicyType = "circuitbreaker"
	RetryPolicy          PolicyType = "retry"
	TimeoutPolicy        PolicyType = "timeout"
)

type PolicyType string

type resiliencyMetrics struct {
	policiesLoadCount *stats.Int64Measure
	executionCount    *stats.Int64Measure

	appID     string
	ctx       context.Context
	enabled   bool
	namespace string
}

func newResiliencyMetrics() *resiliencyMetrics {
	return &resiliencyMetrics{
		policiesLoadCount: stats.Int64(
			"resiliency/loaded",
			"Number of resiliency policies loaded.",
			stats.UnitDimensionless),
		executionCount: stats.Int64(
			"resiliency/count",
			"Number of times a resiliency policyKey has been executed.",
			stats.UnitDimensionless),

		// TODO: how to use correct context
		ctx:     context.Background(),
		enabled: false,
	}
}

// Init registers the resiliency metrics views.
func (m *resiliencyMetrics) Init(id string, namespace string) error {
	m.enabled = true
	m.appID = id
	m.namespace = namespace
	return view.Register(
		diag_utils.NewMeasureView(m.policiesLoadCount, []tag.Key{appIDKey, resiliencyNameKey, namespaceKey}, view.Count()),
		diag_utils.NewMeasureView(m.executionCount, []tag.Key{appIDKey, resiliencyNameKey, policyKey, namespaceKey}, view.Count()),
	)
}

// PolicyLoaded records metric when policy is loaded.
func (m *resiliencyMetrics) PolicyLoaded(resiliencyName, namespace string) {
	if m.enabled {
		_ = stats.RecordWithTags(
			m.ctx,
			diag_utils.WithTags(appIDKey, m.appID, resiliencyNameKey, resiliencyName, namespaceKey, namespace),
			m.policiesLoadCount.M(1),
		)
	}
}

// PolicyExecuted records metric when policy is executed.
func (m *resiliencyMetrics) PolicyExecuted(resiliencyName string, policy PolicyType) {
	if m.enabled {
		_ = stats.RecordWithTags(
			m.ctx,
			diag_utils.WithTags(appIDKey, m.appID, resiliencyNameKey, resiliencyName, policyKey, string(policy), namespaceKey, m.namespace),
			m.executionCount.M(1),
		)
	}
}
