package ai

import (
	"strings"
	"testing"
)

func TestParseStream_TextDeltas(t *testing.T) {
	body := strings.Join([]string{
		`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
		`data: {"choices":[{"delta":{"content":", world"}}]}`,
		`data: [DONE]`,
		``,
	}, "\n")

	var got strings.Builder
	msg, err := parseStream(strings.NewReader(body), func(d string) { got.WriteString(d) })
	if err != nil {
		t.Fatalf("parseStream: %v", err)
	}
	if msg.Content != "Hello, world" {
		t.Fatalf("content = %q, want %q", msg.Content, "Hello, world")
	}
	if got.String() != "Hello, world" {
		t.Fatalf("streamed = %q, want %q", got.String(), "Hello, world")
	}
	if len(msg.ToolCalls) != 0 {
		t.Fatalf("expected no tool calls, got %d", len(msg.ToolCalls))
	}
}

func TestParseStream_FragmentedToolCall(t *testing.T) {
	// A tool call whose name and arguments arrive across multiple chunks.
	body := strings.Join([]string{
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"get_resource"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"kind\":\"pods\","}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"name\":\"web\"}"}}]}}]}`,
		`data: [DONE]`,
		``,
	}, "\n")

	msg, err := parseStream(strings.NewReader(body), nil)
	if err != nil {
		t.Fatalf("parseStream: %v", err)
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(msg.ToolCalls))
	}
	tc := msg.ToolCalls[0]
	if tc.ID != "call_1" || tc.Function.Name != "get_resource" {
		t.Fatalf("unexpected tool call: %+v", tc)
	}
	args := decodeArgs(tc.Function.Arguments)
	if args["kind"] != "pods" || args["name"] != "web" {
		t.Fatalf("decoded args = %v", args)
	}
}

func TestParseStream_SkipsMalformedFrames(t *testing.T) {
	body := strings.Join([]string{
		`: keep-alive comment`,
		`data: not-json`,
		`data: {"choices":[{"delta":{"content":"ok"}}]}`,
		`data: [DONE]`,
		``,
	}, "\n")

	msg, err := parseStream(strings.NewReader(body), nil)
	if err != nil {
		t.Fatalf("parseStream: %v", err)
	}
	if msg.Content != "ok" {
		t.Fatalf("content = %q, want %q", msg.Content, "ok")
	}
}
