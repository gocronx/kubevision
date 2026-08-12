package ai

import (
	"testing"

	"github.com/gocronx/kubevision/internal/model"
)

type auditCapture struct{ logs []model.AuditLog }

func (a *auditCapture) Record(log model.AuditLog) { a.logs = append(a.logs, log) }

func TestRecordMutationAuditAttributesHumanAndOmitsPayload(t *testing.T) {
	audit := &auditCapture{}
	svc := &Service{audit: audit}
	sess := &pendingSession{
		clusterName: "prod", correlationID: "correlation-1",
		toolCall: ToolCall{Function: FunctionCall{Name: "create_resource"}},
	}
	args := map[string]any{
		"kind": "deployments", "yaml": "metadata:\n  name: web\n  namespace: default\nspec:\n  token: secret",
	}
	svc.recordMutationAudit(sess, Actor{UserID: 42, Username: "alice", ClientIP: "127.0.0.1"}, args, 200, "succeeded", 0)

	if len(audit.logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(audit.logs))
	}
	got := audit.logs[0]
	if got.UserID != 42 || got.Username != "alice" || got.Source != "ai-assistant" {
		t.Fatalf("wrong actor attribution: %+v", got)
	}
	if got.Resource != "deployments" || got.Name != "web" || got.Namespace != "default" {
		t.Fatalf("wrong safe target metadata: %+v", got)
	}
	if got.RequestBody != "" || got.Tool != "create_resource" || got.CorrelationID != "correlation-1" {
		t.Fatalf("payload retained or correlation missing: %+v", got)
	}
}
