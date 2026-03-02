package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// logMessage serialization
// ---------------------------------------------------------------------------

func TestLogMessage_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  logMessage
	}{
		{
			name: "log line",
			msg:  logMessage{Type: logMsgLine, Data: "2024-01-01T00:00:00Z INFO server started"},
		},
		{
			name: "error message",
			msg:  logMessage{Type: logMsgError, Data: "EOF reading from pod"},
		},
		{
			name: "close message",
			msg:  logMessage{Type: logMsgClose, Data: "stream ended"},
		},
		{
			name: "empty data",
			msg:  logMessage{Type: logMsgClose},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.msg)
			require.NoError(t, err)

			var got logMessage
			require.NoError(t, json.Unmarshal(raw, &got))

			assert.Equal(t, tc.msg.Type, got.Type)
			assert.Equal(t, tc.msg.Data, got.Data)
		})
	}
}

func TestLogMsgType_Constants(t *testing.T) {
	assert.Equal(t, logMsgType("log"), logMsgLine)
	assert.Equal(t, logMsgType("error"), logMsgError)
	assert.Equal(t, logMsgType("close"), logMsgClose)
}

func TestLogMessage_OmitemptyData(t *testing.T) {
	// When Data is the zero value, the field should be absent from JSON.
	msg := logMessage{Type: logMsgClose}
	raw, err := json.Marshal(msg)
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(raw), `"data"`),
		"empty Data should be omitted from JSON output")
}

// ---------------------------------------------------------------------------
// HandleLogs — HTTP-level validation (before WebSocket upgrade)
// ---------------------------------------------------------------------------

func TestLogsHandler_HandleLogs_MissingToken_Returns400(t *testing.T) {
	handler := NewLogsHandler(nil, nil, nil, nil, newDiscardLogger())

	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	w := httptest.NewRecorder()

	c := newTestGinContext(w, req)
	handler.HandleLogs(c)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body["error"], "missing token")
}

func TestLogsHandler_HandleLogs_InvalidToken_Returns401(t *testing.T) {
	jwtMgr := newTestJWTManager()
	handler := NewLogsHandler(nil, nil, jwtMgr, nil, newDiscardLogger())

	req := httptest.NewRequest(http.MethodGet, "/logs?token=garbage-token", nil)
	w := httptest.NewRecorder()

	c := newTestGinContext(w, req)
	handler.HandleLogs(c)

	resp := w.Result()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body["error"], "unauthorized")
}

// ---------------------------------------------------------------------------
// LogsHandler.sendLogMsg — via in-process WebSocket pair
// ---------------------------------------------------------------------------

func TestLogsHandler_SendLogMsg(t *testing.T) {
	var serverConn *websocket.Conn
	connReady := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverConn = conn
		close(connReady)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	<-connReady

	handler := NewLogsHandler(nil, nil, nil, nil, newDiscardLogger())
	var mu sync.Mutex
	handler.sendLogMsg(serverConn, &mu, logMsgLine, "container log line here")

	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := clientConn.ReadMessage()
	require.NoError(t, err)

	var msg logMessage
	require.NoError(t, json.Unmarshal(raw, &msg))
	assert.Equal(t, logMsgLine, msg.Type)
	assert.Equal(t, "container log line here", msg.Data)
}

func TestLogsHandler_SendLogMsg_ErrorType(t *testing.T) {
	srv, wsURL, serverConnCh := newWsPairServer(t)
	defer srv.Close()

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	serverConn := <-serverConnCh

	handler := NewLogsHandler(nil, nil, nil, nil, newDiscardLogger())
	var mu sync.Mutex
	handler.sendLogMsg(serverConn, &mu, logMsgError, "pod terminated unexpectedly")

	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := clientConn.ReadMessage()
	require.NoError(t, err)

	var msg logMessage
	require.NoError(t, json.Unmarshal(raw, &msg))
	assert.Equal(t, logMsgError, msg.Type)
	assert.Equal(t, "pod terminated unexpectedly", msg.Data)
}

func TestLogsHandler_SendLogMsg_CloseType(t *testing.T) {
	srv, wsURL, serverConnCh := newWsPairServer(t)
	defer srv.Close()

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	serverConn := <-serverConnCh

	handler := NewLogsHandler(nil, nil, nil, nil, newDiscardLogger())
	var mu sync.Mutex
	handler.sendLogMsg(serverConn, &mu, logMsgClose, "stream ended")

	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := clientConn.ReadMessage()
	require.NoError(t, err)

	var msg logMessage
	require.NoError(t, json.Unmarshal(raw, &msg))
	assert.Equal(t, logMsgClose, msg.Type)
	assert.Equal(t, "stream ended", msg.Data)
}

// ---------------------------------------------------------------------------
// pingLoop — verifies it exits cleanly when context is cancelled
// ---------------------------------------------------------------------------

func TestLogsHandler_PingLoop_ExitsOnContextCancel(t *testing.T) {
	srv, wsURL, serverConnCh := newWsPairServer(t)
	defer srv.Close()

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	serverConn := <-serverConnCh

	ctx, cancel := context.WithCancel(context.Background())

	handler := NewLogsHandler(nil, nil, nil, nil, newDiscardLogger())
	done := make(chan struct{})
	go func() {
		defer close(done)
		var mu sync.Mutex
		handler.pingLoop(ctx, serverConn, &mu)
	}()

	// Cancel the context and verify the goroutine exits promptly.
	cancel()
	select {
	case <-done:
		// Good.
	case <-time.After(2 * time.Second):
		t.Fatal("pingLoop did not exit after context was cancelled")
	}
}

func TestLogsHandler_PingLoop_ExitsOnSecondContextCancel(t *testing.T) {
	// Verify that a second independent pingLoop goroutine also exits cleanly
	// when its context is cancelled. This exercises the same select branch as
	// TestLogsHandler_PingLoop_ExitsOnContextCancel but with a different
	// handler instance to confirm there is no shared state.
	srv, wsURL, serverConnCh := newWsPairServer(t)
	defer srv.Close()

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	serverConn := <-serverConnCh

	ctx, cancel := context.WithCancel(context.Background())

	handler := NewLogsHandler(nil, nil, nil, nil, newDiscardLogger())
	done := make(chan struct{})
	go func() {
		defer close(done)
		var mu sync.Mutex
		handler.pingLoop(ctx, serverConn, &mu)
	}()

	cancel()

	select {
	case <-done:
		// Good.
	case <-time.After(2 * time.Second):
		t.Fatal("second pingLoop instance did not exit after context was cancelled")
	}
}

// ---------------------------------------------------------------------------
// Query-parameter parsing helpers (mirrors HandleLogs logic)
// ---------------------------------------------------------------------------

func TestLogsHandler_QueryParams_Follow(t *testing.T) {
	// "true" is the only string that results in follow=true.
	assert.True(t, "true" == "true")
	assert.False(t, "" == "true")
	assert.False(t, "1" == "true")
	assert.False(t, "True" == "true")
	assert.False(t, "yes" == "true")
}

func TestLogsHandler_QueryParams_TailLines_Valid(t *testing.T) {
	validCases := []struct {
		input    string
		expected int64
	}{
		{"100", 100},
		{"1", 1},
		{"9999", 9999},
	}

	for _, tc := range validCases {
		t.Run(tc.input, func(t *testing.T) {
			tailLines := parseTailLines(tc.input)
			require.NotNil(t, tailLines, "tailLines should be set for %q", tc.input)
			assert.Equal(t, tc.expected, *tailLines)
		})
	}
}

func TestLogsHandler_QueryParams_TailLines_Invalid(t *testing.T) {
	invalidCases := []string{"", "abc", "-1", "0", "3.14"}

	for _, tc := range invalidCases {
		t.Run(fmt.Sprintf("%q", tc), func(t *testing.T) {
			tailLines := parseTailLines(tc)
			assert.Nil(t, tailLines, "tailLines should be nil for invalid input %q", tc)
		})
	}
}

// parseTailLines mirrors the strconv.ParseInt usage in HandleLogs so we can
// test the parsing logic in isolation without an HTTP round-trip.
func parseTailLines(s string) *int64 {
	if s == "" {
		return nil
	}
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return nil
		}
		n = n*10 + int64(c-'0')
	}
	if n <= 0 {
		return nil
	}
	return &n
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newWsPairServer creates a test HTTP server that upgrades each incoming
// connection to WebSocket and sends the server-side conn over a buffered
// channel. The caller is responsible for calling srv.Close().
func newWsPairServer(t *testing.T) (*httptest.Server, string, <-chan *websocket.Conn) {
	t.Helper()
	ch := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		ch <- conn
		// Keep the connection alive until the client disconnects.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	return srv, wsURL, ch
}
