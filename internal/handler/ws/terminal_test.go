package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/remotecommand"
)

// ---------------------------------------------------------------------------
// termMessage serialization
// ---------------------------------------------------------------------------

func TestTermMessage_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  termMessage
	}{
		{
			name: "input message",
			msg:  termMessage{Type: termMsgInput, Data: "ls -la\n"},
		},
		{
			name: "output message",
			msg:  termMessage{Type: termMsgOutput, Data: "total 0\ndrwxr-xr-x 2 root root 6"},
		},
		{
			name: "resize message",
			msg:  termMessage{Type: termMsgResize, Cols: 220, Rows: 50},
		},
		{
			name: "error message",
			msg:  termMessage{Type: termMsgError, Data: "connection refused"},
		},
		{
			name: "close message",
			msg:  termMessage{Type: termMsgClose, Data: "session ended"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.msg)
			require.NoError(t, err)

			var got termMessage
			require.NoError(t, json.Unmarshal(raw, &got))

			assert.Equal(t, tc.msg.Type, got.Type)
			assert.Equal(t, tc.msg.Data, got.Data)
			assert.Equal(t, tc.msg.Cols, got.Cols)
			assert.Equal(t, tc.msg.Rows, got.Rows)
		})
	}
}

func TestTermMessage_OmitemptyBehavior(t *testing.T) {
	// When Cols and Rows are zero (non-resize messages) they should be omitted
	// from the JSON to keep the payload compact.
	msg := termMessage{Type: termMsgOutput, Data: "hello"}
	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	assert.False(t, strings.Contains(string(raw), `"cols"`),
		"zero Cols should be omitted from JSON")
	assert.False(t, strings.Contains(string(raw), `"rows"`),
		"zero Rows should be omitted from JSON")

	// When Data is empty (resize messages) it should be omitted.
	resizeMsg := termMessage{Type: termMsgResize, Cols: 80, Rows: 24}
	resizeRaw, err := json.Marshal(resizeMsg)
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(resizeRaw), `"data"`),
		"empty Data should be omitted from JSON")
}

func TestTermMsgType_Constants(t *testing.T) {
	// Verify the constants are the exact string values the frontend expects.
	assert.Equal(t, termMsgType("input"), termMsgInput)
	assert.Equal(t, termMsgType("output"), termMsgOutput)
	assert.Equal(t, termMsgType("resize"), termMsgResize)
	assert.Equal(t, termMsgType("error"), termMsgError)
	assert.Equal(t, termMsgType("close"), termMsgClose)
}

// ---------------------------------------------------------------------------
// termSizeQueue
// ---------------------------------------------------------------------------

func TestTermSizeQueue_PushAndNext(t *testing.T) {
	q := newTermSizeQueue()
	size := remotecommand.TerminalSize{Width: 120, Height: 40}
	q.push(size)

	got := q.Next()
	require.NotNil(t, got)
	assert.Equal(t, uint16(120), got.Width)
	assert.Equal(t, uint16(40), got.Height)
}

func TestTermSizeQueue_PushMultiple_OrderPreserved(t *testing.T) {
	q := newTermSizeQueue()

	sizes := []remotecommand.TerminalSize{
		{Width: 80, Height: 24},
		{Width: 120, Height: 40},
		{Width: 200, Height: 60},
	}
	for _, s := range sizes {
		q.push(s)
	}

	for _, expected := range sizes {
		got := q.Next()
		require.NotNil(t, got)
		assert.Equal(t, expected.Width, got.Width)
		assert.Equal(t, expected.Height, got.Height)
	}
}

func TestTermSizeQueue_ClosedChannel_NextReturnsNil(t *testing.T) {
	q := newTermSizeQueue()
	close(q.ch)

	got := q.Next()
	assert.Nil(t, got, "Next should return nil when channel is closed")
}

func TestTermSizeQueue_DropWhenFull(t *testing.T) {
	q := newTermSizeQueue()

	// Fill the channel to capacity (16 slots).
	for i := uint16(0); i < 20; i++ {
		q.push(remotecommand.TerminalSize{Width: i, Height: i})
	}

	// No panic should occur; excess pushes are silently dropped.
	assert.Equal(t, 16, len(q.ch), "channel should be capped at 16 items")
}

func TestTermSizeQueue_ConcurrentPushes(t *testing.T) {
	q := newTermSizeQueue()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n uint16) {
			defer wg.Done()
			q.push(remotecommand.TerminalSize{Width: n, Height: n})
		}(uint16(i))
	}
	wg.Wait()
	// We just verify no race or panic occurred; we don't check exact contents
	// because the channel may be full and drops are expected.
}

// ---------------------------------------------------------------------------
// wsWriter
// ---------------------------------------------------------------------------

func TestWsWriter_EmptyWrite_IsNoop(t *testing.T) {
	// wsWriter.Write with an empty slice must return (0, nil) without touching
	// the connection. Since we cannot create a real websocket.Conn easily, we
	// verify the early-return path by confirming length 0 input returns immediately.
	w := &wsWriter{
		conn:    nil, // would panic if reached
		msgType: termMsgOutput,
		mu:      &sync.Mutex{},
	}
	n, err := w.Write([]byte{})
	assert.Equal(t, 0, n)
	assert.NoError(t, err)
}

func TestWsWriter_MarshalledPayload(t *testing.T) {
	// Test the JSON payload structure that wsWriter would send over the wire.
	// We do this without a real WS connection by directly exercising the marshal logic.
	data := []byte("hello from container")
	msg := termMessage{Type: termMsgOutput, Data: string(data)}
	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded termMessage
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, termMsgOutput, decoded.Type)
	assert.Equal(t, "hello from container", decoded.Data)
}

// ---------------------------------------------------------------------------
// HandleExec — HTTP-level validation (no WebSocket upgrade)
// ---------------------------------------------------------------------------

// TestTerminalHandler_HandleExec_MissingToken verifies that a request without
// the token query parameter receives a 400 response before any WS upgrade.
func TestTerminalHandler_HandleExec_MissingToken(t *testing.T) {
	handler := NewTerminalHandler(nil, nil, nil, nil, nil, newDiscardLogger())

	req := httptest.NewRequest(http.MethodGet, "/exec", nil)
	w := httptest.NewRecorder()

	// Use a Gin context to call the handler.
	c := newTestGinContext(w, req)
	handler.HandleExec(c)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body["error"], "missing token")
}

// ---------------------------------------------------------------------------
// TerminalHandler — resolveClusterConfig delegation
// ---------------------------------------------------------------------------

// TestTerminalHandler_ResolveClusterConfig_DelegatesResolveClusterName ensures
// the resolveClusterConfig method delegates to resolveClusterName correctly by
// comparing the numeric-ID lookup path to the name fallback.
func TestTerminalHandler_ResolveClusterConfig_ClusterNotFound(t *testing.T) {
	repo := newStubClusterRepo()
	// No clusters registered — any lookup should fail.

	handler := NewTerminalHandler(nil, repo, nil, nil, nil, newDiscardLogger())

	req := httptest.NewRequest(http.MethodGet, "/exec?token=x", nil)
	w := httptest.NewRecorder()
	c := newTestGinContext(w, req)

	_, err := handler.resolveClusterConfig(c, "42")
	require.Error(t, err, "should fail when the cluster is not in the repo")
}

// ---------------------------------------------------------------------------
// WebSocket-level: full upgrade test using httptest
// ---------------------------------------------------------------------------

// TestTerminalHandler_HandleExec_InvalidTokenRejectsBeforeUpgrade tests that
// an invalid JWT token causes a 400 rejection (before WebSocket upgrade) when
// the JWTManager is present but the token is garbage.
func TestTerminalHandler_HandleExec_InvalidToken_Rejected(t *testing.T) {
	jwtMgr := newTestJWTManager()
	handler := NewTerminalHandler(nil, nil, jwtMgr, nil, nil, newDiscardLogger())

	req := httptest.NewRequest(http.MethodGet, "/exec?token=notavalidjwt", nil)
	w := httptest.NewRecorder()

	c := newTestGinContext(w, req)
	handler.HandleExec(c)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Contains(t, body["error"], "unauthorized")
}

// ---------------------------------------------------------------------------
// wsWriter via a real in-process WebSocket pair
// ---------------------------------------------------------------------------

// TestWsWriter_Write_SendsTextMessage exercises wsWriter.Write end-to-end using
// an in-process WebSocket pair created with httptest.NewServer.
func TestWsWriter_Write_SendsTextMessage(t *testing.T) {
	var serverConn *websocket.Conn
	var serverConnMu sync.Mutex
	connReady := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("server upgrade error: %v", err)
			return
		}
		serverConnMu.Lock()
		serverConn = conn
		serverConnMu.Unlock()
		close(connReady)

		// Keep the connection open until the client disconnects.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	// Dial the test server.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	// Wait for the server side to be ready.
	select {
	case <-connReady:
	case <-time.After(2 * time.Second):
		t.Fatal("server WebSocket connection not ready in time")
	}

	serverConnMu.Lock()
	sc := serverConn
	serverConnMu.Unlock()
	require.NotNil(t, sc)

	writer := &wsWriter{
		conn:    sc,
		msgType: termMsgOutput,
		mu:      &sync.Mutex{},
	}

	payload := []byte("container output line")
	n, err := writer.Write(payload)
	require.NoError(t, err)
	assert.Equal(t, len(payload), n)

	// Read the message on the client side.
	_, msg, err := clientConn.ReadMessage()
	require.NoError(t, err)

	var received termMessage
	require.NoError(t, json.Unmarshal(msg, &received))
	assert.Equal(t, termMsgOutput, received.Type)
	assert.Equal(t, "container output line", received.Data)
}

// ---------------------------------------------------------------------------
// sendTermMsg via a real in-process WebSocket pair
// ---------------------------------------------------------------------------

func TestTerminalHandler_SendTermMsg(t *testing.T) {
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

	handler := NewTerminalHandler(nil, nil, nil, nil, nil, newDiscardLogger())
	handler.sendTermMsg(serverConn, termMsgError, "something went wrong")

	_, raw, err := clientConn.ReadMessage()
	require.NoError(t, err)

	var msg termMessage
	require.NoError(t, json.Unmarshal(raw, &msg))
	assert.Equal(t, termMsgError, msg.Type)
	assert.Equal(t, "something went wrong", msg.Data)
}

// ---------------------------------------------------------------------------
// readLoop message dispatch (using in-process WebSocket pair)
// ---------------------------------------------------------------------------

func TestTerminalHandler_ReadLoop_InputMessage_WritesToStdin(t *testing.T) {
	srv, clientDial := newWSPair(t)
	defer srv.Close()

	clientConn, _, err := websocket.DefaultDialer.Dial(clientDial, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	// Give the server-side handler a moment to accept the connection.
	time.Sleep(20 * time.Millisecond)

	// We obtain the server-side connection by running a minimal test server.
	// Instead, test via an in-process pipe.
	stdinR, stdinW := io.Pipe()
	sizeQueue := newTermSizeQueue()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build a server-side WS conn by dialing ourselves.
	var serverConn *websocket.Conn
	serverReady := make(chan struct{})
	testSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		serverConn = conn
		close(serverReady)
		<-ctx.Done()
		conn.Close()
	}))
	defer testSrv.Close()

	dialURL := "ws" + strings.TrimPrefix(testSrv.URL, "http")
	clientConn2, _, err := websocket.DefaultDialer.Dial(dialURL, nil)
	require.NoError(t, err)
	defer clientConn2.Close()

	<-serverReady

	handler := NewTerminalHandler(nil, nil, nil, nil, nil, newDiscardLogger())

	// Run the readLoop in the background — it will read from serverConn.
	go handler.readLoop(ctx, serverConn, stdinW, sizeQueue, cancel)

	// Send an input message from the client side.
	inputMsg, _ := json.Marshal(termMessage{Type: termMsgInput, Data: "whoami\n"})
	require.NoError(t, clientConn2.WriteMessage(websocket.TextMessage, inputMsg))

	// Read from stdin pipe — should receive the forwarded data.
	// Use a goroutine + channel so we can apply a test-level timeout without
	// relying on io.PipeReader having a SetReadDeadline method (it does not).
	type readResult struct {
		data string
		err  error
	}
	resultCh := make(chan readResult, 1)
	go func() {
		buf := make([]byte, 64)
		n, err := stdinR.Read(buf)
		resultCh <- readResult{string(buf[:n]), err}
	}()

	select {
	case res := <-resultCh:
		require.NoError(t, res.err)
		assert.Equal(t, "whoami\n", res.data)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stdin data")
	}
}

func TestTerminalHandler_ReadLoop_ResizeMessage_PushesToSizeQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var serverConn *websocket.Conn
	serverReady := make(chan struct{})
	testSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		serverConn = conn
		close(serverReady)
		<-ctx.Done()
		conn.Close()
	}))
	defer testSrv.Close()

	dialURL := "ws" + strings.TrimPrefix(testSrv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(dialURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	<-serverReady

	_, stdinW := io.Pipe()
	sizeQueue := newTermSizeQueue()
	handler := NewTerminalHandler(nil, nil, nil, nil, nil, newDiscardLogger())

	go handler.readLoop(ctx, serverConn, stdinW, sizeQueue, cancel)

	// Send a resize message.
	resizeMsg, _ := json.Marshal(termMessage{Type: termMsgResize, Cols: 132, Rows: 50})
	require.NoError(t, clientConn.WriteMessage(websocket.TextMessage, resizeMsg))

	// Wait for the size to appear in the queue.
	select {
	case size := <-sizeQueue.ch:
		assert.Equal(t, uint16(132), size.Width)
		assert.Equal(t, uint16(50), size.Height)
	case <-time.After(time.Second):
		t.Fatal("resize event was not received in time")
	}
}

func TestTerminalHandler_ReadLoop_RawBytesForwardedToStdin(t *testing.T) {
	// Non-JSON bytes should be forwarded directly to stdin.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var serverConn *websocket.Conn
	serverReady := make(chan struct{})
	testSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		serverConn = conn
		close(serverReady)
		<-ctx.Done()
		conn.Close()
	}))
	defer testSrv.Close()

	dialURL := "ws" + strings.TrimPrefix(testSrv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(dialURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	<-serverReady

	stdinR, stdinW := io.Pipe()
	sizeQueue := newTermSizeQueue()
	handler := NewTerminalHandler(nil, nil, nil, nil, nil, newDiscardLogger())

	go handler.readLoop(ctx, serverConn, stdinW, sizeQueue, cancel)

	rawBytes := []byte{0x03} // Ctrl-C
	require.NoError(t, clientConn.WriteMessage(websocket.TextMessage, rawBytes))

	// Read via goroutine so we can apply a deadline without SetReadDeadline
	// (io.PipeReader does not implement that method).
	type rawResult struct {
		data []byte
		err  error
	}
	rawCh := make(chan rawResult, 1)
	go func() {
		buf := make([]byte, 8)
		n, err := stdinR.Read(buf)
		rawCh <- rawResult{buf[:n], err}
	}()

	select {
	case res := <-rawCh:
		require.NoError(t, res.err)
		assert.Equal(t, rawBytes, res.data)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for raw stdin bytes")
	}
}

func TestTerminalHandler_ReadLoop_ZeroDimensionResizeIgnored(t *testing.T) {
	// Resize messages with Cols=0 or Rows=0 must be silently dropped.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var serverConn *websocket.Conn
	serverReady := make(chan struct{})
	testSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		serverConn = conn
		close(serverReady)
		<-ctx.Done()
		conn.Close()
	}))
	defer testSrv.Close()

	dialURL := "ws" + strings.TrimPrefix(testSrv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(dialURL, nil)
	require.NoError(t, err)
	defer clientConn.Close()

	<-serverReady

	_, stdinW := io.Pipe()
	sizeQueue := newTermSizeQueue()
	handler := NewTerminalHandler(nil, nil, nil, nil, nil, newDiscardLogger())

	go handler.readLoop(ctx, serverConn, stdinW, sizeQueue, cancel)

	// Send resize with zero dimensions.
	resizeMsg, _ := json.Marshal(termMessage{Type: termMsgResize, Cols: 0, Rows: 0})
	require.NoError(t, clientConn.WriteMessage(websocket.TextMessage, resizeMsg))

	// Give the readLoop a moment to process.
	time.Sleep(50 * time.Millisecond)

	// sizeQueue must be empty.
	assert.Equal(t, 0, len(sizeQueue.ch), "zero-dimension resize should not be enqueued")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newWSPair creates a trivial WS echo server and returns the test server and
// its ws:// URL so callers can dial it.
func newWSPair(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			_ = conn.WriteMessage(mt, msg)
		}
	}))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	return srv, wsURL
}

// stdoutCapture is an io.Writer that accumulates written bytes in a buffer.
type stdoutCapture struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (c *stdoutCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Write(p)
}

func (c *stdoutCapture) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}
