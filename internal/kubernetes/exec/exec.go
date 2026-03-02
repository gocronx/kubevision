package exec

import (
	"context"
	"fmt"
	"io"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// Session handles exec/attach sessions to containers via WebSocket.
// It bridges the browser WebSocket connection to a Kubernetes pod exec SPDY stream.
type Session struct {
	clientset  kubernetes.Interface
	restConfig *rest.Config
}

// NewSession creates a new exec Session from the given cluster REST config.
func NewSession(restConfig *rest.Config) (*Session, error) {
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes clientset: %w", err)
	}
	return &Session{
		clientset:  clientset,
		restConfig: restConfig,
	}, nil
}

// ExecOptions holds the parameters for a pod exec call.
type ExecOptions struct {
	// Ctx is the parent context. Cancelling it terminates the exec stream.
	Ctx       context.Context
	Namespace string
	PodName   string
	Container string
	Command   []string
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	TTY       bool
	// TerminalSizeQueue receives terminal resize events from the client.
	TerminalSizeQueue remotecommand.TerminalSizeQueue
}

// Exec starts a remote command in a pod container using the Kubernetes SPDY exec protocol.
// It streams stdin/stdout/stderr between the caller and the remote process and
// forwards terminal resize events via TerminalSizeQueue.
// This call blocks until the remote command exits or an error occurs.
func (s *Session) Exec(opts ExecOptions) error {
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}
	if len(opts.Command) == 0 {
		opts.Command = []string{"/bin/sh"}
	}

	req := s.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(opts.PodName).
		Namespace(opts.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: opts.Container,
			Command:   opts.Command,
			Stdin:     opts.Stdin != nil,
			Stdout:    opts.Stdout != nil,
			Stderr:    opts.Stderr != nil,
			TTY:       opts.TTY,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(s.restConfig, http.MethodPost, req.URL())
	if err != nil {
		return fmt.Errorf("create SPDY executor: %w", err)
	}

	return executor.StreamWithContext(opts.Ctx, remotecommand.StreamOptions{
		Stdin:             opts.Stdin,
		Stdout:            opts.Stdout,
		Stderr:            opts.Stderr,
		Tty:               opts.TTY,
		TerminalSizeQueue: opts.TerminalSizeQueue,
	})
}

// LogOptions holds the parameters for streaming pod logs.
type LogOptions struct {
	Namespace  string
	PodName    string
	Container  string
	Follow     bool
	Previous   bool
	TailLines  *int64
	Timestamps bool
}

// StreamLogs streams logs for the specified pod container to the given writer.
// When Follow is true the call blocks until the stream closes or an error occurs.
func StreamLogs(ctx context.Context, clientset kubernetes.Interface, opts LogOptions, writer io.Writer) error {
	podLogOpts := &corev1.PodLogOptions{
		Container:  opts.Container,
		Follow:     opts.Follow,
		Previous:   opts.Previous,
		TailLines:  opts.TailLines,
		Timestamps: opts.Timestamps,
	}

	req := clientset.CoreV1().Pods(opts.Namespace).GetLogs(opts.PodName, podLogOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("open log stream: %w", err)
	}
	defer stream.Close()

	_, err = io.Copy(writer, stream)
	return err
}

// NewClientset creates a kubernetes.Interface from the given rest.Config.
func NewClientset(restConfig *rest.Config) (kubernetes.Interface, error) {
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes clientset: %w", err)
	}
	return clientset, nil
}
