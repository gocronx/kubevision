package ai

import (
	"testing"

	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
)

func TestApproveActionConsumesOwnedSession(t *testing.T) {
	service := &Service{sessions: newSessionStore()}
	id, ok := service.sessions.save(&pendingSession{userID: 7, clusterID: 3, clusterName: "demo", toolCall: ToolCall{ID: "call-1", Function: FunctionCall{Name: "apply_resource"}}})
	if !ok {
		t.Fatal("failed to save pending session")
	}
	approved, err := service.ApproveAction(id, Actor{UserID: 7, Username: "operator"})
	if err != nil {
		t.Fatal(err)
	}
	if approved.ClusterID != 3 || approved.ToolCall.ID != "call-1" {
		t.Fatalf("unexpected approved action: %#v", approved)
	}
	if _, err := service.ApproveAction(id, Actor{UserID: 7}); !bizerr.Is(err, bizerr.CodeValidation) {
		t.Fatalf("expected consumed session to be rejected, got %v", err)
	}
}
