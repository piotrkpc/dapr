package diagnostics_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	resiliency_v1alpha "github.com/dapr/dapr/pkg/apis/resiliency/v1alpha1"
	diag "github.com/dapr/dapr/pkg/diagnostics"
	"github.com/dapr/dapr/pkg/resiliency"
	"github.com/dapr/kit/logger"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResiliencyMonitoring(t *testing.T) {
	// require := require.New(t)
	resiliencyConf := createTestResiliencyConfig()

	t.Run("resiliency/loaded", func(t *testing.T) {
		diag.InitMetrics("fakeID", "fakeRuntimeNamespace")
		_ = resiliency.FromConfigurations(logger.NewLogger("fake-logger"), &resiliencyConf)

		rows, err := view.RetrieveData("resiliency/loaded")
		require.NoError(t, err)
		require.Equal(t, 1, len(rows))
		require.False(t, rows[0].Data.StartTime().IsZero())
		require.Contains(t, rows[0].Tags, tag.Tag{Key: tag.MustNewKey("app_id"), Value: "fakeID"})
		require.Contains(t, rows[0].Tags, tag.Tag{Key: tag.MustNewKey("name"), Value: "testResiliency"})
		require.Contains(t, rows[0].Tags, tag.Tag{Key: tag.MustNewKey("namespace"), Value: "testNamespace"})
	})

	t.Run("resiliency/count", func(t *testing.T) {
		diag.InitMetrics("fakeID", "fakeRuntimeNamespace")

		r := resiliency.FromConfigurations(logger.NewLogger("fake-logger"), &resiliencyConf)
		_ = r.EndpointPolicy(context.TODO(), "appB", "fakeEndpoint")

		rows, err := view.RetrieveData("resiliency/count")
		require.NoError(t, err)
		require.Equal(t, 3, len(rows))
		requireTagExist(t, rows, "app_id", "fakeID")
		requireTagExist(t, rows, "name", "testTimeout")
		requireTagExist(t, rows, "name", "testRetry")
		requireTagExist(t, rows, "name", "testCB")
		requireTagExist(t, rows, "namespace", "fakeRuntimeNamespace")
		requireTagExist(t, rows, "policy", "timeout")
		requireTagExist(t, rows, "policy", "retry")
		requireTagExist(t, rows, "policy", "circuitbreaker")

	})

}

func requireTagExist(t *testing.T, rows []*view.Row, key string, value string) {
	// TODO: refactor this
	t.Helper()
	var found bool
	aTag := tag.Tag{tag.MustNewKey(key), value}
outerLoop:
	for _, row := range rows {
		for _, jTag := range row.Tags {
			if reflect.DeepEqual(aTag, jTag) {
				found = true
				break outerLoop
			}
		}
	}
	require.True(t, found, fmt.Sprintf("did not found tag (%s, %s) in rows:", key, value), rows)
}

func createTestResiliencyConfig() resiliency_v1alpha.Resiliency {
	return resiliency_v1alpha.Resiliency{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testResiliency",
			Namespace: "testNamespace",
		},
		Spec: resiliency_v1alpha.ResiliencySpec{
			Policies: resiliency_v1alpha.Policies{
				Timeouts: map[string]string{
					"testTimeout": "5s",
				},
				Retries: map[string]resiliency_v1alpha.Retry{
					"testRetry": {
						Policy:     "constant",
						Duration:   "5s",
						MaxRetries: 10,
					},
				},
				CircuitBreakers: map[string]resiliency_v1alpha.CircuitBreaker{
					"testCB": {
						Interval:    "8s",
						Timeout:     "45s",
						Trip:        "consecutiveFailures > 8",
						MaxRequests: 1,
					},
				},
			},
			Targets: resiliency_v1alpha.Targets{
				Apps: map[string]resiliency_v1alpha.EndpointPolicyNames{
					"appB": {
						Timeout:                 "testTimeout",
						Retry:                   "testRetry",
						CircuitBreaker:          "testCB",
						CircuitBreakerCacheSize: 100,
					},
				},
				Actors: map[string]resiliency_v1alpha.ActorPolicyNames{
					"myActorType": {
						Timeout:                 "general",
						Retry:                   "general",
						CircuitBreaker:          "general",
						CircuitBreakerScope:     "both",
						CircuitBreakerCacheSize: 5000,
					},
				},
				Components: map[string]resiliency_v1alpha.ComponentPolicyNames{
					"statestore1": {
						Outbound: resiliency_v1alpha.PolicyNames{
							Timeout:        "general",
							Retry:          "general",
							CircuitBreaker: "general",
						},
					},
				},
			},
		},
	}
}
