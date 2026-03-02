package ws

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/kubevision/kubevision/internal/auth"
	"github.com/kubevision/kubevision/internal/kubernetes/cluster"
	kubexec "github.com/kubevision/kubevision/internal/kubernetes/exec"
	"github.com/kubevision/kubevision/internal/repository"
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
	logger         *zap.Logger
}

// NewTerminalHandler creates a new TerminalHandler.
func NewTerminalHandler(
	clusterManager *cluster.Manager,
	clusterRepo repository.ClusterRepo,
	jwtManager *auth.JWTManager,
	userRepo repository.UserRepo,
	logger *zap.Logger,
) *TerminalHandler {
	return &TerminalHandler{
		clusterManager: clusterManager,
		clusterRepo:    clusterRepo,
		jwtManager:     jwtManager,
		userRepo:       userRepo,
		logger:         logger,
	}
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
	if _, err := h.authenticateToken(c, tokenStr); err != nil {
		h.writeErrorAndClose(c, "unauthorized: "+err.Error())
		return
	}

	// --- Route parameters ---
	clusterIDStr := c.Param("id")
	namespace := c.Param("namespace")
	podName := c.Param("name")
	container := c.Query("container")
	shell := c.DefaultQuery("command", "sh")

	// Resolve numeric cluster DB ID to the cluster name key used by the manager.
	restConfig, err := h.resolveClusterConfig(c, clusterIDStr)
	if err != nil {
		h.writeErrorAndClose(c, "cluster not found: "+err.Error())
		return
	}

	// --- Upgrade to WebSocket ---
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("terminal ws upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

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
	sizeQueue := newTermSizeQueue()

	// wsReader pipes WebSocket input messages to the exec stdin.
	stdinReader, stdinWriter := io.Pipe()
	defer stdinWriter.Close()

	// stdoutWriter writes exec output back to the WebSocket.
	stdoutWriter := &wsWriter{conn: conn, msgType: termMsgOutput, mu: &sync.Mutex{}}

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
		Stdout:            stdoutWriter,
		Stderr:            stdoutWriter, // merge stderr into the same stream
		TTY:               true,
		TerminalSizeQueue: sizeQueue,
	})

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

// authenticateToken validates a JWT token string against the database.
func (h *TerminalHandler) authenticateToken(c *gin.Context, tokenStr string) (*auth.TokenClaims, error) {
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
	return claims, nil
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
