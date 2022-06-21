package diagnostics_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	resiliency_v1alpha "github.com/dapr/dapr/pkg/apis/resiliency/v1alpha1"
	diag "github.com/dapr/dapr/pkg/diagnostics"
	"github.com/dapr/dapr/pkg/resiliency"
	"github.com/dapr/kit/logger"
)

const (
	testResiliencyName       = "testResiliency"
	testResiliencyNamespace  = "testNamespace"
	resiliencyCountViewName  = "resiliency/count"
	resiliencyLoadedViewName = "resiliency/loaded"
	testAppID                = "fakeID"
)

func TestResiliencyMonitoring(t *testing.T) {
	// TODO: refactor to table tests

	t.Run(resiliencyLoadedViewName, func(t *testing.T) {
		t.Cleanup(func() {
			view.Unregister(view.Find(resiliencyCountViewName))
		})
		diag.InitMetrics(testAppID, "fakeRuntimeNamespace")
		_ = resiliency.FromConfigurations(
			logger.NewLogger("fake-logger"),
			createTestResiliencyConfig(),
		)

		rows, err := view.RetrieveData(resiliencyLoadedViewName)

		require.NoError(t, err)
		require.Equal(t, 1, len(rows))
		requireTagExist(t, rows, "app_id", testAppID)
		requireTagExist(t, rows, "name", testResiliencyName)
		requireTagExist(t, rows, "namespace", testResiliencyNamespace)
	})

	t.Run(resiliencyCountViewName, func(t *testing.T) {
		t.Run("EndpointPolicy", func(t *testing.T) {
			t.Cleanup(func() {
				view.Unregister(view.Find(resiliencyCountViewName))
			})
			diag.InitMetrics(testAppID, "fakeRuntimeNamespace")

			r := resiliency.FromConfigurations(logger.NewLogger("fake-logger"), createTestResiliencyConfig())
			_ = r.EndpointPolicy(context.TODO(), "appB", "fakeEndpoint")

			rows, err := view.RetrieveData(resiliencyCountViewName)
			require.NoError(t, err)
			require.Equal(t, 3, len(rows))
			requireTagExist(t, rows, "app_id", testAppID)
			requireTagExist(t, rows, "name", testResiliencyName)
			requireTagExist(t, rows, "namespace", testResiliencyNamespace)
			requireTagExist(t, rows, "policy", "timeout")
			requireTagExist(t, rows, "policy", "retry")
			requireTagExist(t, rows, "policy", "circuitbreaker")
		})

		t.Run("ActorPreLockPolicy", func(t *testing.T) {
			t.Cleanup(func() {
				view.Unregister(view.Find(resiliencyCountViewName))
			})
			diag.InitMetrics(testAppID, "fakeRuntimeNamespace")

			r := resiliency.FromConfigurations(logger.NewLogger("fake-logger"), createTestResiliencyConfig())
			_ = r.ActorPreLockPolicy(context.TODO(), "myActorType", "fakeId")

			rows, err := view.RetrieveData(resiliencyCountViewName)
			require.NoError(t, err)
			require.Equal(t, 2, len(rows))
			requireTagExist(t, rows, "app_id", testAppID)
			requireTagExist(t, rows, "name", testResiliencyName)
			requireTagExist(t, rows, "namespace", testResiliencyNamespace)
			requireTagExist(t, rows, "policy", "retry")
			requireTagExist(t, rows, "policy", "circuitbreaker")
		})
		t.Run("ActorPostLockPolicy", func(t *testing.T) {
			t.Cleanup(func() {
				view.Unregister(view.Find(resiliencyCountViewName))
			})
			diag.InitMetrics(testAppID, "fakeRuntimeNamespace")

			r := resiliency.FromConfigurations(logger.NewLogger("fake-logger"), createTestResiliencyConfig())
			_ = r.ActorPostLockPolicy(context.TODO(), "myActorType", "fakeId")

			rows, err := view.RetrieveData(resiliencyCountViewName)
			require.NoError(t, err)
			require.Equal(t, 1, len(rows))
			requireTagExist(t, rows, "app_id", testAppID)
			requireTagExist(t, rows, "name", testResiliencyName)
			requireTagExist(t, rows, "namespace", testResiliencyNamespace)
			requireTagExist(t, rows, "policy", "timeout")
		})
		t.Run("ComponentOutboundPolicy", func(t *testing.T) {
			t.Cleanup(func() {
				view.Unregister(view.Find(resiliencyCountViewName))
			})
			diag.InitMetrics(testAppID, "fakeRuntimeNamespace")

			r := resiliency.FromConfigurations(logger.NewLogger("fake-logger"), createTestResiliencyConfig())
			_ = r.ComponentOutboundPolicy(context.TODO(), "statestore1")

			rows, err := view.RetrieveData(resiliencyCountViewName)
			require.NoError(t, err)
			require.Equal(t, 3, len(rows))
			requireTagExist(t, rows, "app_id", testAppID)
			requireTagExist(t, rows, "name", testResiliencyName)
			requireTagExist(t, rows, "namespace", testResiliencyNamespace)
			requireTagExist(t, rows, "policy", "timeout")
			requireTagExist(t, rows, "policy", "retry")
			requireTagExist(t, rows, "policy", "circuitbreaker")
		})
		t.Run("ComponentInboundPolicy", func(t *testing.T) {
			t.Cleanup(func() {
				view.Unregister(view.Find(resiliencyCountViewName))
			})
			diag.InitMetrics(testAppID, "fakeRuntimeNamespace")

			r := resiliency.FromConfigurations(logger.NewLogger("fake-logger"), createTestResiliencyConfig())
			_ = r.ComponentInboundPolicy(context.TODO(), "statestore1")

			rows, err := view.RetrieveData(resiliencyCountViewName)
			require.NoError(t, err)
			require.Equal(t, 3, len(rows))
			requireTagExist(t, rows, "app_id", testAppID)
			requireTagExist(t, rows, "name", testResiliencyName)
			requireTagExist(t, rows, "namespace", testResiliencyNamespace)
			requireTagExist(t, rows, "policy", "timeout")
			requireTagExist(t, rows, "policy", "retry")
			requireTagExist(t, rows, "policy", "circuitbreaker")
		})
	})
}

func requireTagExist(t *testing.T, rows []*view.Row, key string, value string) {
	t.Helper()
	var found bool
	aTag := tag.Tag{Key: tag.MustNewKey(key), Value: value}
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

func createTestResiliencyConfig() *resiliency_v1alpha.Resiliency {
	return testResiliencyConfig(testResiliencyName, testResiliencyNamespace, "appB", "myActorType", "statestore1")
}

func testResiliencyConfig(resilencyName, resiliencyNamespace, appName, actorType, storeName string) *resiliency_v1alpha.Resiliency {
	return &resiliency_v1alpha.Resiliency{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resilencyName,
			Namespace: resiliencyNamespace,
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
					appName: {
						Timeout:                 "testTimeout",
						Retry:                   "testRetry",
						CircuitBreaker:          "testCB",
						CircuitBreakerCacheSize: 100,
					},
				},
				Actors: map[string]resiliency_v1alpha.ActorPolicyNames{
					actorType: {
						Timeout:                 "testTimeout",
						Retry:                   "testRetry",
						CircuitBreaker:          "testCB",
						CircuitBreakerScope:     "both",
						CircuitBreakerCacheSize: 5000,
					},
				},
				Components: map[string]resiliency_v1alpha.ComponentPolicyNames{
					storeName: {
						Outbound: resiliency_v1alpha.PolicyNames{
							Timeout:        "testTimeout",
							Retry:          "testRetry",
							CircuitBreaker: "testCB",
						},
						Inbound: resiliency_v1alpha.PolicyNames{
							Timeout:        "testTimeout",
							Retry:          "testRetry",
							CircuitBreaker: "testCB",
						},
					},
				},
			},
		},
	}
}
