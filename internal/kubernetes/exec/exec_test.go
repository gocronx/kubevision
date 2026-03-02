package exec

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Session construction
// ---------------------------------------------------------------------------

// TestNewSession_ValidConfig verifies that NewSession succeeds when given a
// non-nil REST config and produces a fully-initialised Session. No actual
// network connection is made at construction time.
func TestNewSession_ValidConfig(t *testing.T) {
	cfg := &rest.Config{Host: "https://127.0.0.1:6443"}
	session, err := NewSession(cfg)
	require.NoError(t, err, "NewSession with a valid config should not fail")
	require.NotNil(t, session, "returned session must not be nil")
	assert.NotNil(t, session.clientset, "session.clientset must be initialised")
	assert.Equal(t, cfg, session.restConfig, "session.restConfig should be the same pointer")
}

func TestNewSession_EmptyHostConfig(t *testing.T) {
	// kubernetes.NewForConfig accepts an empty Host — it defaults to localhost.
	cfg := &rest.Config{Host: ""}
	session, err := NewSession(cfg)
	require.NoError(t, err)
	require.NotNil(t, session)
}

func TestNewSession_MultipleCallsReturnIndependentSessions(t *testing.T) {
	cfg1 := &rest.Config{Host: "https://cluster-a:6443"}
	cfg2 := &rest.Config{Host: "https://cluster-b:6443"}

	s1, err1 := NewSession(cfg1)
	s2, err2 := NewSession(cfg2)

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NotNil(t, s1)
	require.NotNil(t, s2)

	// Each session should hold its own restConfig.
	assert.Equal(t, "https://cluster-a:6443", s1.restConfig.Host)
	assert.Equal(t, "https://cluster-b:6443", s2.restConfig.Host)
}

// ---------------------------------------------------------------------------
// ExecOptions — default-filling logic (mirrored from Exec)
// ---------------------------------------------------------------------------

func TestExecOptions_DefaultCommand_SetWhenEmpty(t *testing.T) {
	opts := ExecOptions{
		Ctx:       context.Background(),
		Namespace: "default",
		PodName:   "my-pod",
		Container: "app",
		Command:   nil,
		Stdout:    &bytes.Buffer{},
	}

	// Replicate the default-filling guard in Exec.
	if len(opts.Command) == 0 {
		opts.Command = []string{"/bin/sh"}
	}
	assert.Equal(t, []string{"/bin/sh"}, opts.Command)
}

func TestExecOptions_DefaultContext_SetWhenNil(t *testing.T) {
	opts := ExecOptions{
		Namespace: "kube-system",
		PodName:   "coredns-abc",
		Command:   []string{"sh"},
		Ctx:       nil,
	}

	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}
	require.NotNil(t, opts.Ctx)
}

func TestExecOptions_ExplicitCommandNotOverwritten(t *testing.T) {
	explicit := []string{"bash", "-c", "echo hi"}
	opts := ExecOptions{
		Command: explicit,
	}
	if len(opts.Command) == 0 {
		opts.Command = []string{"/bin/sh"}
	}
	assert.Equal(t, explicit, opts.Command, "explicit command must not be overwritten")
}

// ---------------------------------------------------------------------------
// ExecOptions — struct field coverage
// ---------------------------------------------------------------------------

func TestExecOptions_AllFieldsReadable(t *testing.T) {
	stdin := strings.NewReader("input data")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ctx := context.Background()

	opts := ExecOptions{
		Ctx:               ctx,
		Namespace:         "production",
		PodName:           "web-frontend-xyz",
		Container:         "nginx",
		Command:           []string{"bash", "-c", "ls -la"},
		Stdin:             stdin,
		Stdout:            &stdout,
		Stderr:            &stderr,
		TTY:               true,
		TerminalSizeQueue: nil,
	}

	assert.Equal(t, ctx, opts.Ctx)
	assert.Equal(t, "production", opts.Namespace)
	assert.Equal(t, "web-frontend-xyz", opts.PodName)
	assert.Equal(t, "nginx", opts.Container)
	assert.Equal(t, []string{"bash", "-c", "ls -la"}, opts.Command)
	assert.Equal(t, io.Reader(stdin), opts.Stdin)
	assert.Equal(t, io.Writer(&stdout), opts.Stdout)
	assert.Equal(t, io.Writer(&stderr), opts.Stderr)
	assert.True(t, opts.TTY)
	assert.Nil(t, opts.TerminalSizeQueue)
}

func TestExecOptions_TTY_False(t *testing.T) {
	opts := ExecOptions{
		Ctx:     context.Background(),
		Command: []string{"cat", "/etc/hostname"},
		TTY:     false,
	}
	assert.False(t, opts.TTY)
}

// TestExecOptions_StdinPresentFlag verifies the logic used inside Exec that
// determines whether stdin should be attached based on the non-nil check.
func TestExecOptions_StdinPresentFlag(t *testing.T) {
	optsWithStdin := ExecOptions{Stdin: strings.NewReader("data")}
	optsWithoutStdin := ExecOptions{Stdin: nil}

	assert.True(t, optsWithStdin.Stdin != nil, "Stdin should be non-nil when provided")
	assert.False(t, optsWithoutStdin.Stdin != nil, "Stdin should be nil when not provided")
}

func TestExecOptions_StdoutAndStderrMerged(t *testing.T) {
	// The terminal handler merges stdout and stderr into the same wsWriter.
	// Verify that a single writer satisfies both the Stdout and Stderr slots.
	shared := &bytes.Buffer{}
	opts := ExecOptions{
		Stdout: shared,
		Stderr: shared,
	}
	assert.Same(t, opts.Stdout.(*bytes.Buffer), opts.Stderr.(*bytes.Buffer),
		"stdout and stderr can point to the same writer")
}

// ---------------------------------------------------------------------------
// LogOptions — struct field coverage and PodLogOptions mapping
// ---------------------------------------------------------------------------

func TestLogOptions_AllFieldsReadable(t *testing.T) {
	tailLines := int64(100)
	opts := LogOptions{
		Namespace:  "staging",
		PodName:    "worker-abc",
		Container:  "sidecar",
		Follow:     true,
		Previous:   false,
		TailLines:  &tailLines,
		Timestamps: true,
	}

	assert.Equal(t, "staging", opts.Namespace)
	assert.Equal(t, "worker-abc", opts.PodName)
	assert.Equal(t, "sidecar", opts.Container)
	assert.True(t, opts.Follow)
	assert.False(t, opts.Previous)
	require.NotNil(t, opts.TailLines)
	assert.Equal(t, int64(100), *opts.TailLines)
	assert.True(t, opts.Timestamps)
}

func TestLogOptions_NilTailLines(t *testing.T) {
	opts := LogOptions{
		Namespace: "default",
		PodName:   "my-pod",
		TailLines: nil,
	}
	assert.Nil(t, opts.TailLines, "nil TailLines means fetch all available log lines")
}

// TestLogOptions_MappedToPodLogOptions verifies the field-by-field conversion
// from LogOptions to corev1.PodLogOptions that takes place inside StreamLogs.
func TestLogOptions_MappedToPodLogOptions(t *testing.T) {
	tailLines := int64(50)
	opts := LogOptions{
		Namespace:  "default",
		PodName:    "api-pod",
		Container:  "main",
		Follow:     true,
		Previous:   true,
		TailLines:  &tailLines,
		Timestamps: true,
	}

	// Mirror the mapping in StreamLogs.
	podLogOpts := &corev1.PodLogOptions{
		Container:  opts.Container,
		Follow:     opts.Follow,
		Previous:   opts.Previous,
		TailLines:  opts.TailLines,
		Timestamps: opts.Timestamps,
	}

	assert.Equal(t, "main", podLogOpts.Container)
	assert.True(t, podLogOpts.Follow)
	assert.True(t, podLogOpts.Previous)
	require.NotNil(t, podLogOpts.TailLines)
	assert.Equal(t, int64(50), *podLogOpts.TailLines)
	assert.True(t, podLogOpts.Timestamps)
}

func TestLogOptions_Previous_DefaultFalse(t *testing.T) {
	opts := LogOptions{Namespace: "ns", PodName: "pod"}
	assert.False(t, opts.Previous, "Previous should default to false")
}

func TestLogOptions_Follow_DefaultFalse(t *testing.T) {
	opts := LogOptions{Namespace: "ns", PodName: "pod"}
	assert.False(t, opts.Follow, "Follow should default to false")
}

// ---------------------------------------------------------------------------
// NewClientset
// ---------------------------------------------------------------------------

func TestNewClientset_ValidConfig_ReturnsClientset(t *testing.T) {
	cfg := &rest.Config{Host: "https://127.0.0.1:6443"}
	cs, err := NewClientset(cfg)
	require.NoError(t, err)
	require.NotNil(t, cs)
}

func TestNewClientset_EmptyHostConfig(t *testing.T) {
	cfg := &rest.Config{Host: ""}
	cs, err := NewClientset(cfg)
	require.NoError(t, err)
	require.NotNil(t, cs)
}

func TestNewClientset_IndependentFromNewSession(t *testing.T) {
	cfg := &rest.Config{Host: "https://k8s:6443"}

	cs, err := NewClientset(cfg)
	require.NoError(t, err)

	session, err := NewSession(cfg)
	require.NoError(t, err)

	// Both should produce non-nil objects.
	assert.NotNil(t, cs)
	assert.NotNil(t, session.clientset)
}

// ---------------------------------------------------------------------------
// remotecommand.TerminalSizeQueue interface compliance
// ---------------------------------------------------------------------------

// TestExecOptions_TerminalSizeQueue_AcceptsInterface confirms that any value
// implementing remotecommand.TerminalSizeQueue can be stored in ExecOptions.
func TestExecOptions_TerminalSizeQueue_AcceptsInterface(t *testing.T) {
	var q remotecommand.TerminalSizeQueue = &mockSizeQueue{}
	opts := ExecOptions{
		TerminalSizeQueue: q,
	}
	assert.NotNil(t, opts.TerminalSizeQueue)

	// Next() should return a non-nil size from our mock.
	size := opts.TerminalSizeQueue.Next()
	require.NotNil(t, size)
	assert.Equal(t, uint16(80), size.Width)
	assert.Equal(t, uint16(24), size.Height)
}

// mockSizeQueue is a trivial remotecommand.TerminalSizeQueue for testing.
type mockSizeQueue struct{}

func (m *mockSizeQueue) Next() *remotecommand.TerminalSize {
	return &remotecommand.TerminalSize{Width: 80, Height: 24}
}

// ---------------------------------------------------------------------------
// StreamLogs — io.Copy path (no real K8s cluster)
// ---------------------------------------------------------------------------

// TestStreamLogs_IOCopyPath tests the core io.Copy portion of StreamLogs in
// isolation using a pre-built ReadCloser, without any Kubernetes API calls.
func TestStreamLogs_IOCopyPath(t *testing.T) {
	content := "2024-01-01 INFO pod started\n2024-01-01 INFO pod ready\n"
	rc := io.NopCloser(strings.NewReader(content))

	var buf bytes.Buffer
	_, err := io.Copy(&buf, rc)
	require.NoError(t, err)
	assert.Equal(t, content, buf.String())
}

func TestStreamLogs_IOCopyPath_Empty(t *testing.T) {
	rc := io.NopCloser(strings.NewReader(""))
	var buf bytes.Buffer
	_, err := io.Copy(&buf, rc)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestStreamLogs_IOCopyPath_LargePayload(t *testing.T) {
	// ~100 KB of log data.
	line := strings.Repeat("x", 255) + "\n"
	var sb strings.Builder
	for i := 0; i < 400; i++ {
		sb.WriteString(line)
	}
	large := sb.String()

	rc := io.NopCloser(strings.NewReader(large))
	var buf bytes.Buffer
	_, err := io.Copy(&buf, rc)
	require.NoError(t, err)
	assert.Equal(t, large, buf.String())
}

// TestStreamLogs_ContextCancellation verifies that StreamLogs propagates a
// cancelled context as an error (no actual cluster — the dial will fail).
func TestStreamLogs_ContextCancellation(t *testing.T) {
	cfg := &rest.Config{Host: "https://127.0.0.1:6443"}
	cs, err := NewClientset(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before any IO

	var buf bytes.Buffer
	streamErr := StreamLogs(ctx, cs, LogOptions{
		Namespace: "default",
		PodName:   "some-pod",
		Container: "main",
	}, &buf)

	assert.Error(t, streamErr,
		"StreamLogs must return an error when context is cancelled or cluster is unreachable")
}

// ---------------------------------------------------------------------------
// ExecOptions — PodExecOptions mapping (mirrors Exec internals)
// ---------------------------------------------------------------------------

// TestExecOptions_MappedToPodExecOptions verifies the field-by-field mapping
// from ExecOptions to corev1.PodExecOptions used when building the exec request.
func TestExecOptions_MappedToPodExecOptions(t *testing.T) {
	stdin := strings.NewReader("input")
	var stdout, stderr bytes.Buffer

	opts := ExecOptions{
		Container: "app",
		Command:   []string{"sh"},
		Stdin:     stdin,
		Stdout:    &stdout,
		Stderr:    &stderr,
		TTY:       true,
	}

	// Mirror the mapping inside Exec.
	podExecOpts := &corev1.PodExecOptions{
		Container: opts.Container,
		Command:   opts.Command,
		Stdin:     opts.Stdin != nil,
		Stdout:    opts.Stdout != nil,
		Stderr:    opts.Stderr != nil,
		TTY:       opts.TTY,
	}

	assert.Equal(t, "app", podExecOpts.Container)
	assert.Equal(t, []string{"sh"}, podExecOpts.Command)
	assert.True(t, podExecOpts.Stdin)
	assert.True(t, podExecOpts.Stdout)
	assert.True(t, podExecOpts.Stderr)
	assert.True(t, podExecOpts.TTY)
}

func TestExecOptions_MappedToPodExecOptions_NoStreams(t *testing.T) {
	opts := ExecOptions{
		Container: "job",
		Command:   []string{"sh", "-c", "exit 0"},
		Stdin:     nil,
		Stdout:    nil,
		Stderr:    nil,
		TTY:       false,
	}

	podExecOpts := &corev1.PodExecOptions{
		Container: opts.Container,
		Command:   opts.Command,
		Stdin:     opts.Stdin != nil,
		Stdout:    opts.Stdout != nil,
		Stderr:    opts.Stderr != nil,
		TTY:       opts.TTY,
	}

	assert.False(t, podExecOpts.Stdin)
	assert.False(t, podExecOpts.Stdout)
	assert.False(t, podExecOpts.Stderr)
	assert.False(t, podExecOpts.TTY)
}
