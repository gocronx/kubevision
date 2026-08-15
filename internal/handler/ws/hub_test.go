package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gocronx/kubevision/internal/kubernetes/informer"
	"go.uber.org/zap/zaptest"
)

func TestNewHub(t *testing.T) {
	logger := zaptest.NewLogger(t)
	hub := NewHub(logger)
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}
	if hub.clients == nil {
		t.Error("clients map should be initialized")
	}
	if hub.broadcast == nil {
		t.Error("broadcast channel should be initialized")
	}
	if hub.register == nil {
		t.Error("register channel should be initialized")
	}
	if hub.unregister == nil {
		t.Error("unregister channel should be initialized")
	}
}

func TestHub_StopIsIdempotentAndStopsRun(t *testing.T) {
	hub := NewHub(zaptest.NewLogger(t))
	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()

	hub.Stop()
	hub.Stop()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("hub Run did not stop")
	}
}

func TestHub_OnResourceEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	hub := NewHub(logger)

	event := informer.ResourceEvent{
		ClusterID: "prod",
		Resource:  "pods",
		Name:      "nginx",
		Namespace: "default",
		Type:      "ADDED",
	}

	// Should not block even without a consumer.
	hub.OnResourceEvent(event)

	// Read from the broadcast channel.
	select {
	case msg := <-hub.broadcast:
		var got informer.ResourceEvent
		if err := json.Unmarshal(msg, &got); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}
		if got.ClusterID != "prod" {
			t.Errorf("expected clusterID 'prod', got %q", got.ClusterID)
		}
		if got.Resource != "pods" {
			t.Errorf("expected resource 'pods', got %q", got.Resource)
		}
	default:
		t.Error("expected message on broadcast channel")
	}
}

func TestHub_OnResourceEvent_ChannelFull(t *testing.T) {
	logger := zaptest.NewLogger(t)
	hub := NewHub(logger)

	event := informer.ResourceEvent{
		ClusterID: "prod",
		Resource:  "pods",
		Name:      "nginx",
	}

	// Fill the broadcast channel.
	for i := 0; i < 1024; i++ {
		hub.OnResourceEvent(event)
	}

	// This should not block — the event is dropped.
	hub.OnResourceEvent(event)
}

func TestClient_isSubscribed_NoTopics(t *testing.T) {
	client := &Client{
		topics: make(map[string]bool),
	}

	msg, _ := json.Marshal(map[string]string{
		"clusterId": "prod",
		"resource":  "pods",
	})

	if !client.isSubscribed(msg) {
		t.Error("client with no subscriptions should receive all messages")
	}
}

func TestClient_isSubscribed_MatchingTopic(t *testing.T) {
	client := &Client{
		topics: map[string]bool{
			"prod:pods": true,
		},
	}

	msg, _ := json.Marshal(map[string]string{
		"clusterId": "prod",
		"resource":  "pods",
	})

	if !client.isSubscribed(msg) {
		t.Error("client subscribed to 'prod:pods' should receive pods events")
	}
}

func TestClient_isSubscribed_NonMatchingTopic(t *testing.T) {
	client := &Client{
		topics: map[string]bool{
			"prod:deployments": true,
		},
	}

	msg, _ := json.Marshal(map[string]string{
		"clusterId": "prod",
		"resource":  "pods",
	})

	if client.isSubscribed(msg) {
		t.Error("client subscribed to 'prod:deployments' should NOT receive pods events")
	}
}

func TestClient_isSubscribed_InvalidJSON(t *testing.T) {
	client := &Client{
		topics: map[string]bool{
			"prod:pods": true,
		},
	}

	// Invalid JSON should return true (deliver unparseable messages).
	if !client.isSubscribed([]byte(`{invalid`)) {
		t.Error("unparseable messages should be delivered")
	}
}
