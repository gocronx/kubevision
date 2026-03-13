package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/kubernetes/cluster"
	kubexec "github.com/gocronx/kubevision/internal/kubernetes/exec"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
	"github.com/gocronx/kubevision/internal/service"
	"k8s.io/client-go/tools/remotecommand"
)

// termMsgType classifies messages on the terminal WebSocket channel.
type termMsgType string

const (
	termMsgInput  termMsgType = "input"
	termMsgOutput termMsgType = "output"
	termMsgResize termMsgType = "resize"
	termMsgError  termMsgType = "error"
	termMsgClose  termMsgType = "close"
)

// termMessage is the JSON envelope for all terminal WebSocket messages.
type termMessage struct {
	Type termMsgType `json:"type"`
	// Data carries raw terminal bytes (base64-free; plain UTF-8 is fine for xterm.js).
	Data string `json:"data,omitempty"`
	// Cols and Rows are set only for resize messages.
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// TerminalHandler handles WebSocket-based pod exec (interactive terminal) sessions.
type TerminalHandler struct {
	clusterManager *cluster.Manager
	clusterRepo    repository.ClusterRepo
	jwtManager     *auth.JWTManager
	userRepo       repository.UserRepo
	roleRepo       repository.RoleRepo
	sessionService *service.TerminalSessionService
	logger         *zap.Logger
}

// NewTerminalHandler creates a new TerminalHandler.
func NewTerminalHandler(
	clusterManager *cluster.Manager,
	clusterRepo repository.ClusterRepo,
	jwtManager *auth.JWTManager,
	userRepo repository.UserRepo,
	roleRepo repository.RoleRepo,
	logger *zap.Logger,
) *TerminalHandler {
	return &TerminalHandler{
		clusterManager: clusterManager,
		clusterRepo:    clusterRepo,
		jwtManager:     jwtManager,
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		logger:         logger,
	}
}

// WithSessionService attaches the TerminalSessionService so recordings are
// persisted when sessions end.
func (h *TerminalHandler) WithSessionService(svc *service.TerminalSessionService) *TerminalHandler {
	h.sessionService = svc
	return h
}

// HandleExec is the Gin handler for:
//
//	GET /api/v1/clusters/:id/namespaces/:namespace/pods/:name/exec
//
// Query parameters:
//
//	token     - JWT access token (required; browsers cannot send custom headers on WS)
//	container - container name (optional; first container used when absent)
//	command   - shell to launch, e.g. "bash", "sh" (default: "sh")
func (h *TerminalHandler) HandleExec(c *gin.Context) {
	// --- Authentication via query-param token ---
	tokenStr := c.Query("token")
	if tokenStr == "" {
		h.writeErrorAndClose(c, "missing token query parameter")
		return
	}
	authResult, err := h.authenticateToken(c, tokenStr)
	if err != nil {
		h.writeErrorAndClose(c, "unauthorized: "+err.Error())
		return
	}
	claims := authResult.Claims

	// --- RBAC: require pods:exec permission ---
	if h.roleRepo != nil {
		if permErr := checkWSPermission(c.Request.Context(), h.roleRepo, authResult.UserRole, "pods", "exec"); permErr != nil {
			h.writeErrorAndClose(c, "permission denied")
			return
		}
	}

	// --- Route parameters ---
	clusterIDStr := c.Param("id")
	namespace := c.Param("namespace")
	podName := c.Param("name")
	container := c.Query("container")
	shellParam := c.Query("command") // empty means auto-detect

	// Resolve numeric cluster DB ID to the cluster name key used by the manager.
	restConfig, err := h.resolveClusterConfig(c, clusterIDStr)
	if err != nil {
		h.writeErrorAndClose(c, "cluster not found: "+err.Error())
		return
	}

	// Resolve cluster name for recording metadata.
	clusterName, _ := resolveClusterName(c.Request.Context(), clusterIDStr, h.clusterRepo)

	// --- Upgrade to WebSocket ---
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("terminal ws upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	// Detect a working shell. If the user specified one, use it directly.
	// Otherwise probe common shells so distroless / minimal images still work.
	shell := shellParam
	if shell == "" {
		h.sendTermMsg(conn, termMsgOutput, "\x1b[90mDetecting available shell...\x1b[0m\r\n")
		detected := h.detectShell(restConfig, namespace, podName, container)
		if detected == "" {
			h.sendTermMsg(conn, termMsgError, "no shell found in container (tried /bin/bash, /bin/sh, bash, sh) — this container may be distroless")
			h.sendTermMsg(conn, termMsgClose, "session ended")
			return
		}
		shell = detected
	}

	h.logger.Info("terminal session started",
		zap.String("pod", podName),
		zap.String("namespace", namespace),
		zap.String("container", container),
		zap.String("shell", shell),
	)

	// --- Build exec session ---
	session, err := kubexec.NewSession(restConfig)
	if err != nil {
		h.sendTermMsg(conn, termMsgError, "failed to create exec session: "+err.Error())
		return
	}

	// Create a context that we cancel when the WebSocket closes or the exec exits.
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// sizeQueue feeds terminal resize events to the SPDY stream.
	// Closing it on exit unblocks any goroutine waiting in Next().
	sizeQueue := newTermSizeQueue()
	defer sizeQueue.Close()

	// wsReader pipes WebSocket input messages to the exec stdin.
	stdinReader, stdinWriter := io.Pipe()
	defer stdinWriter.Close()

	// recordingBuf accumulates asciinema v2 output events for later persistence.
	var recordingBuf bytes.Buffer
	sessionStartTime := time.Now()

	// Write asciinema v2 header.
	header := fmt.Sprintf(`{"version":2,"width":220,"height":50,"timestamp":%d,"title":"%s/%s"}`,
		sessionStartTime.Unix(), namespace, podName)
	recordingBuf.WriteString(header + "\n")

	// recordingWriter intercepts output bytes and appends asciinema v2 events.
	recWriter := &recordingWriter{
		conn:      conn,
		msgType:   termMsgOutput,
		mu:        &sync.Mutex{},
		startTime: sessionStartTime,
		recordBuf: &recordingBuf,
		recordMu:  &sync.Mutex{},
	}

	// Start reading WebSocket messages (input + resize) in a goroutine.
	go h.readLoop(ctx, conn, stdinWriter, sizeQueue, cancel)

	// Run the exec. This blocks until the remote process exits.
	execErr := session.Exec(kubexec.ExecOptions{
		Ctx:               ctx,
		Namespace:         namespace,
		PodName:           podName,
		Container:         container,
		Command:           []string{shell},
		Stdin:             stdinReader,
		Stdout:            recWriter,
		Stderr:            recWriter, // merge stderr into the same stream
		TTY:               true,
		TerminalSizeQueue: sizeQueue,
	})

	sessionDuration := time.Since(sessionStartTime)

	if execErr != nil && ctx.Err() == nil {
		h.logger.Warn("terminal exec ended with error",
			zap.String("pod", podName),
			zap.Error(execErr),
		)
		h.sendTermMsg(conn, termMsgError, execErr.Error())
	}

	// Notify the client that the session is closed.
	h.sendTermMsg(conn, termMsgClose, "session ended")
	h.logger.Info("terminal session ended", zap.String("pod", podName))

	// Persist the recording asynchronously to avoid blocking the WebSocket close.
	if h.sessionService != nil && recordingBuf.Len() > 0 {
		recording := recordingBuf.String()
		userID := claims.UserID
		go func() {
			saveCtx, saveCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer saveCancel()
			sess := &model.TerminalSession{
				UserID:     userID,
				Cluster:    clusterName,
				Namespace:  namespace,
				Pod:        podName,
				Container:  container,
				Recording:  recording,
				DurationMs: sessionDuration.Milliseconds(),
			}
			if err := h.sessionService.Save(saveCtx, sess); err != nil {
				h.logger.Warn("failed to save terminal session recording", zap.Error(err))
			}
		}()
	}
}

// readLoop reads messages from the browser WebSocket and dispatches them as either
// stdin data or resize events. It runs until the context is cancelled or the
// WebSocket is closed.
func (h *TerminalHandler) readLoop(
	ctx context.Context,
	conn *websocket.Conn,
	stdinWriter *io.PipeWriter,
	sizeQueue *termSizeQueue,
	cancel context.CancelFunc,
) {
	defer func() {
		stdinWriter.Close()
		cancel()
	}()

	conn.SetReadLimit(32 * 1024)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
				h.logger.Debug("terminal ws read error", zap.Error(err))
			}
			return
		}

		var msg termMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			// Not JSON — treat as raw stdin (for compatibility with xterm.js send).
			_, _ = stdinWriter.Write(raw)
			continue
		}

		switch msg.Type {
		case termMsgInput:
			_, _ = stdinWriter.Write([]byte(msg.Data))
		case termMsgResize:
			if msg.Cols > 0 && msg.Rows > 0 {
				sizeQueue.push(remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows})
			}
		}
	}
}

// sendTermMsg writes a terminal message to the WebSocket connection.
func (h *TerminalHandler) sendTermMsg(conn *websocket.Conn, msgType termMsgType, data string) {
	msg := termMessage{Type: msgType, Data: data}
	raw, _ := json.Marshal(msg)
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	_ = conn.WriteMessage(websocket.TextMessage, raw)
}

// writeErrorAndClose sends an HTTP 400 before the WebSocket upgrade (used for
// early validation failures where upgrade has not yet happened).
func (h *TerminalHandler) writeErrorAndClose(c *gin.Context, msg string) {
	h.logger.Warn("terminal handler rejected request", zap.String("reason", msg))
	c.JSON(400, gin.H{"error": msg})
}

// resolveClusterConfig resolves a cluster ID string (numeric DB ID or name)
// to its REST config via the cluster manager.
func (h *TerminalHandler) resolveClusterConfig(c *gin.Context, clusterIDStr string) (*k8sRestConfig, error) {
	clusterName, err := resolveClusterName(c.Request.Context(), clusterIDStr, h.clusterRepo)
	if err != nil {
		return nil, err
	}
	return h.clusterManager.RESTConfig(clusterName)
}

// detectShell probes the container for a usable shell by attempting a quick
// exec of each candidate. Returns the first shell that exits successfully.
// Falls back to "sh" if none work (let the real session surface the error).
func (h *TerminalHandler) detectShell(restConfig *k8sRestConfig, namespace, pod, container string) string {
	candidates := []string{"/bin/bash", "/bin/sh", "bash", "sh"}
	for _, sh := range candidates {
		session, err := kubexec.NewSession(restConfig)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = session.Exec(kubexec.ExecOptions{
			Ctx:       ctx,
			Namespace: namespace,
			PodName:   pod,
			Container: container,
			Command:   []string{sh, "-c", "true"},
			Stdout:    io.Discard,
			Stderr:    io.Discard,
			TTY:       false,
		})
		cancel()
		if err == nil {
			h.logger.Info("detected shell", zap.String("pod", pod), zap.String("shell", sh))
			return sh
		}
	}
	h.logger.Warn("no shell detected in container", zap.String("pod", pod))
	return ""
}

// terminalAuthResult bundles the validated JWT claims with the user's
// current role as stored in the database (may differ from the JWT claim
// if the role was changed after the token was issued).
type terminalAuthResult struct {
	Claims   *auth.TokenClaims
	UserRole string
}

// authenticateToken validates a JWT token string against the database.
func (h *TerminalHandler) authenticateToken(c *gin.Context, tokenStr string) (*terminalAuthResult, error) {
	claims, err := h.jwtManager.ParseToken(tokenStr)
	if err != nil {
		return nil, err
	}
	user, err := h.userRepo.GetByID(c.Request.Context(), claims.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, errAccountDisabled
	}
	if claims.TokenVersion != user.TokenVersion {
		return nil, errTokenRevoked
	}
	return &terminalAuthResult{Claims: claims, UserRole: user.Role}, nil
}

// --------------------------------------------------------------------------
// termSizeQueue implements remotecommand.TerminalSizeQueue using a buffered
// channel so resize events never block the WebSocket read loop.
// --------------------------------------------------------------------------

type termSizeQueue struct {
	ch chan remotecommand.TerminalSize
}

func newTermSizeQueue() *termSizeQueue {
	return &termSizeQueue{ch: make(chan remotecommand.TerminalSize, 16)}
}

func (q *termSizeQueue) push(size remotecommand.TerminalSize) {
	select {
	case q.ch <- size:
	default:
		// Drop if the channel is full to avoid blocking the caller.
	}
}

// Close closes the underlying channel so that any goroutine blocked in Next()
// receives a nil and can exit cleanly.
func (q *termSizeQueue) Close() {
	close(q.ch)
}

// Next implements remotecommand.TerminalSizeQueue. It blocks until a size is
// available, which is the expected behaviour for the SPDY executor.
func (q *termSizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-q.ch
	if !ok {
		return nil
	}
	return &size
}

// --------------------------------------------------------------------------
// wsWriter is an io.Writer that encodes each Write call as a terminal output
// message and sends it on the WebSocket connection.
// --------------------------------------------------------------------------

type wsWriter struct {
	conn    *websocket.Conn
	msgType termMsgType
	mu      *sync.Mutex
}

func (w *wsWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	msg := termMessage{Type: w.msgType, Data: string(p)}
	raw, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := w.conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		return 0, err
	}
	return len(p), nil
}

// --------------------------------------------------------------------------
// recordingWriter wraps wsWriter and also appends each output chunk to an
// asciinema v2 recording buffer.
// --------------------------------------------------------------------------

// asciinemaEvent is the JSON line format for asciinema v2 recordings.
// Each event is: [elapsed_seconds, "o", "data"]
type asciinemaEvent = [3]interface{}

type recordingWriter struct {
	conn      *websocket.Conn
	msgType   termMsgType
	mu        *sync.Mutex
	startTime time.Time
	recordBuf *bytes.Buffer
	recordMu  *sync.Mutex
}

func (w *recordingWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Send to WebSocket.
	msg := termMessage{Type: w.msgType, Data: string(p)}
	raw, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}
	w.mu.Lock()
	_ = w.conn.SetWriteDeadline(time.Now().Add(writeWait))
	wsErr := w.conn.WriteMessage(websocket.TextMessage, raw)
	w.mu.Unlock()
	if wsErr != nil {
		return 0, wsErr
	}

	// Append to recording buffer as asciinema v2 event.
	elapsed := time.Since(w.startTime).Seconds()
	event := asciinemaEvent{elapsed, "o", string(p)}
	eventJSON, _ := json.Marshal(event)

	w.recordMu.Lock()
	w.recordBuf.Write(eventJSON)
	w.recordBuf.WriteByte('\n')
	w.recordMu.Unlock()

	return len(p), nil
}
