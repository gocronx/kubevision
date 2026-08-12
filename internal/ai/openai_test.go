package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClientListModels(t *testing.T) {
	var authorization string
	client := NewClient(Config{BaseURL: "https://provider.example/v1/", APIKey: "secret"})
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		authorization = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %q, want /v1/models", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"model-b"},{"id":"model-a"}]}`)),
		}, nil
	})
	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if authorization != "Bearer secret" {
		t.Fatalf("Authorization = %q", authorization)
	}
	if len(models) != 2 || models[0].ID != "model-b" || models[1].ID != "model-a" {
		t.Fatalf("models = %#v", models)
	}
}

func TestClientListModelsProviderError(t *testing.T) {
	client := NewClient(Config{BaseURL: "https://provider.example/v1", APIKey: "bad"})
	client.httpClient.Transport = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":"invalid key"}`)),
		}, nil
	})

	_, err := client.ListModels(context.Background())
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("error = %v, want provider status", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

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
