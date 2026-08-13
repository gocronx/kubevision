package ai

import (
	"context"
	"testing"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
)

// fakeLLM returns pre-scripted assistant turns, one per Stream call.
type fakeLLM struct {
	turns []Message
	calls int
}

func (f *fakeLLM) Stream(_ context.Context, _ []Message, _ []Tool, onContent func(string)) (Message, error) {
	m := f.turns[f.calls]
	f.calls++
	if m.Content != "" && onContent != nil {
		onContent(m.Content)
	}
	return m, nil
}

// fakeK8s is a no-op K8sResourceRepo returning canned data for read tools.
type fakeK8s struct{}

func (fakeK8s) List(context.Context, string, string, string, repository.ListOptions) (*repository.ResourceList, error) {
	return &repository.ResourceList{Items: []repository.Resource{{Kind: "Pod", Name: "web", Namespace: "default"}}, Total: 1}, nil
}
func (fakeK8s) Get(context.Context, string, string, string, string) (*repository.Resource, error) {
	return &repository.Resource{Kind: "Pod", Name: "web", Namespace: "default", Raw: map[string]any{"kind": "Pod"}}, nil
}
func (fakeK8s) Create(context.Context, string, string, string, map[string]any) (*repository.Resource, error) {
	return &repository.Resource{Kind: "Pod", Name: "web", Namespace: "default"}, nil
}
func (fakeK8s) Update(context.Context, string, string, string, string, map[string]any) (*repository.Resource, error) {
	return &repository.Resource{Kind: "Pod", Name: "web", Namespace: "default"}, nil
}
func (fakeK8s) Delete(context.Context, string, string, string, string) error { return nil }
func (fakeK8s) Patch(context.Context, string, string, string, string, []byte) (*repository.Resource, error) {
	return &repository.Resource{Kind: "Deployment", Name: "web", Namespace: "default"}, nil
}
func (fakeK8s) DryRunCreate(context.Context, string, string, string, map[string]any) (*repository.Resource, error) {
	return nil, nil
}
func (fakeK8s) DryRunUpdate(context.Context, string, string, string, string, map[string]any) (*repository.Resource, *repository.Resource, error) {
	return nil, nil, nil
}

func collectRun(role string, turns []Message) (*run, *[]SSEEvent) {
	var events []SSEEvent
	r := &run{
		client:      &fakeLLM{turns: turns},
		exec:        &executor{k8s: fakeK8s{}, clusterName: "prod"},
		authz:       newAuthorizer(&stubRoleRepo{roles: map[string]*model.Role{"viewer": roleWith("pods:get", "pods:list")}}),
		sessions:    newSessionStore(),
		role:        role,
		clusterName: "prod",
		emit:        func(ev SSEEvent) { events = append(events, ev) },
	}
	return r, &events
}

func eventNames(events []SSEEvent) []string {
	names := make([]string, len(events))
	for i, e := range events {
		names[i] = e.Event
	}
	return names
}

func TestLoop_ReadToolThenAnswer(t *testing.T) {
	turns := []Message{
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "c1", Function: FunctionCall{Name: "list_resources", Arguments: `{"kind":"pods"}`}}}},
		{Role: "assistant", Content: "There is 1 pod."},
	}
	r, events := collectRun("admin", turns)
	r.loop(context.Background(), []Message{{Role: "user", Content: "list pods"}})

	got := eventNames(*events)
	// tool_call, tool_result, message, done
	want := []string{EventToolCall, EventToolResult, EventMessage, EventDone}
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] = %s, want %s (all: %v)", i, got[i], want[i], got)
		}
	}
}

func TestLoop_RetriesEmptyFinalResponse(t *testing.T) {
	turns := []Message{
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "c1", Function: FunctionCall{Name: "list_resources", Arguments: `{"kind":"pods"}`}}}},
		{Role: "assistant"},
		{Role: "assistant", Content: "Inspection complete."},
	}
	r, events := collectRun("admin", turns)
	r.loop(context.Background(), []Message{{Role: "user", Content: "inspect pods"}})

	got := eventNames(*events)
	want := []string{EventToolCall, EventToolResult, EventMessage, EventDone}
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] = %s, want %s (all: %v)", i, got[i], want[i], got)
		}
	}
}

func TestLoop_ReportsRepeatedEmptyFinalResponse(t *testing.T) {
	turns := []Message{{Role: "assistant"}, {Role: "assistant"}}
	r, events := collectRun("admin", turns)
	r.loop(context.Background(), []Message{{Role: "user", Content: "inspect pods"}})

	got := eventNames(*events)
	if len(got) != 1 || got[0] != EventError {
		t.Fatalf("events = %v, want [%s]", got, EventError)
	}
}

func TestLoop_MutationPausesForApproval(t *testing.T) {
	turns := []Message{
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "c1", Function: FunctionCall{Name: "delete_resource", Arguments: `{"kind":"pods","name":"web","namespace":"default"}`}}}},
	}
	r, events := collectRun("admin", turns)
	r.loop(context.Background(), []Message{{Role: "user", Content: "delete web"}})

	got := eventNames(*events)
	want := []string{EventToolCall, EventActionRequired}
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	// The loop must pause: no done event, and a session must be saved.
	last := (*events)[len(*events)-1]
	data := last.Data.(map[string]any)
	if data["session_id"] == "" || data["session_id"] == nil {
		t.Fatal("action_required must carry a session_id")
	}
}

func TestLoop_RBACDenialFeedsForbidden(t *testing.T) {
	turns := []Message{
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "c1", Function: FunctionCall{Name: "delete_resource", Arguments: `{"kind":"pods","name":"web","namespace":"default"}`}}}},
		{Role: "assistant", Content: "You lack permission to delete pods."},
	}
	r, events := collectRun("viewer", turns)
	r.loop(context.Background(), []Message{{Role: "user", Content: "delete web"}})

	// The denied tool result should be an error, and the loop should continue to
	// a final answer (no action_required, since the mutation was blocked).
	var sawForbidden, sawAction bool
	for _, e := range *events {
		if e.Event == EventToolResult {
			if d, ok := e.Data.(map[string]any); ok {
				if isErr, _ := d["is_error"].(bool); isErr {
					sawForbidden = true
				}
			}
		}
		if e.Event == EventActionRequired {
			sawAction = true
		}
	}
	if !sawForbidden {
		t.Fatal("expected a forbidden tool_result")
	}
	if sawAction {
		t.Fatal("denied mutation must not request approval")
	}
}
