package informer

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestSummarizeCacheSyncTreatsShutdownAsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := summarizeCacheSync(ctx, map[schema.GroupVersionResource]bool{
		{Group: "apps", Version: "v1", Resource: "deployments"}: false,
		{Version: "v1", Resource: "pods"}:                       false,
	})

	if !result.canceled {
		t.Fatal("expected canceled sync result")
	}
	if len(result.failed) != 0 {
		t.Fatalf("shutdown must not report resource failures, got %d", len(result.failed))
	}
}

func TestSummarizeCacheSyncPreservesRealFailures(t *testing.T) {
	failedGVR := schema.GroupVersionResource{Version: "v1", Resource: "pods"}
	result := summarizeCacheSync(context.Background(), map[schema.GroupVersionResource]bool{
		{Group: "apps", Version: "v1", Resource: "deployments"}: true,
		failedGVR: false,
	})

	if result.canceled {
		t.Fatal("active sync must not be marked canceled")
	}
	if result.allSynced {
		t.Fatal("expected partial sync result")
	}
	if len(result.failed) != 1 || result.failed[0] != failedGVR {
		t.Fatalf("unexpected failed resources: %#v", result.failed)
	}
}
