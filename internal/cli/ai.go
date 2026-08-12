package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/gocronx/kubevision/internal/ai"
	"github.com/gocronx/kubevision/internal/kubernetes/cluster"
	"github.com/gocronx/kubevision/internal/kubernetes/resource"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/plugin/prometheus"
)

// AIChat starts the terminal AI assistant. With a prompt argument it answers
// once and exits; with no prompt it opens an interactive REPL.
//
// Usage:
//
//	kubevision ai "why is the web pod crashing in default?"
//	kubevision ai            # interactive
func AIChat(args []string) error {
	fs := flag.NewFlagSet("ai", flag.ContinueOnError)
	kubeconfig := fs.String("kubeconfig", "", "path to kubeconfig (default: $KUBECONFIG or ~/.kube/config)")
	apiKey := fs.String("api-key", "", "LLM API key (default: $API_KEY)")
	baseURL := fs.String("base-url", "", "LLM base URL (default: $API_BASE_URL)")
	modelID := fs.String("model", "", "model id (default: $MODEL_ID)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	key := firstNonEmpty(*apiKey, os.Getenv("API_KEY"))
	if key == "" {
		return fmt.Errorf("no API key: pass --api-key or set API_KEY")
	}
	cfg := ai.Config{
		Enabled:   true,
		APIKey:    key,
		BaseURL:   firstNonEmpty(*baseURL, os.Getenv("API_BASE_URL")),
		Model:     firstNonEmpty(*modelID, os.Getenv("MODEL_ID")),
		MaxTokens: 4096,
	}

	kubeData, err := os.ReadFile(resolveKubeconfig(*kubeconfig))
	if err != nil {
		return fmt.Errorf("read kubeconfig: %w", err)
	}
	clusterName := currentContext(kubeData)

	mgr := cluster.NewManager()
	if err := mgr.Add(clusterName, kubeData); err != nil {
		return fmt.Errorf("connect to cluster: %w", err)
	}
	registry := resource.NewRegistry()

	svc := ai.NewService(
		ai.NewConfigStore(&staticSettingRepo{cfg: cfg}),
		newDirectRepo(mgr, registry),
		mgr,
		&staticClusterRepo{name: clusterName},
		registry,
		&staticRoleRepo{},
		func() (*prometheus.Plugin, bool) { return nil, false },
		nil,
	)

	r := &aiRunner{
		svc:         svc,
		clusterName: clusterName,
		in:          bufio.NewReader(os.Stdin),
		out:         os.Stdout,
		color:       term.IsTerminal(int(os.Stdout.Fd())),
	}

	// One-shot mode: a prompt was given on the command line.
	if prompt := strings.TrimSpace(strings.Join(fs.Args(), " ")); prompt != "" {
		r.ask(context.Background(), prompt)
		return nil
	}

	// Interactive REPL.
	fmt.Fprintf(r.out, "KubeVision AI — cluster %q, model %q. Type 'exit' to quit.\n", clusterName, cfg.Model)
	for {
		fmt.Fprint(r.out, "\n› ")
		line, err := r.in.ReadString('\n')
		if err != nil {
			return nil // EOF (Ctrl-D)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}
		r.ask(context.Background(), line)
	}
}

// aiRunner drives a multi-turn conversation, printing streamed output and
// prompting for approval before any mutation runs.
type aiRunner struct {
	svc         *ai.Service
	clusterName string
	history     []ai.Message
	in          *bufio.Reader
	out         io.Writer
	color       bool
}

func (r *aiRunner) ask(ctx context.Context, text string) {
	r.history = append(r.history, ai.Message{Role: "user", Content: text})

	var assistant strings.Builder
	var pending string // session id of a mutation awaiting approval

	emit := func(ev ai.SSEEvent) {
		data, _ := ev.Data.(map[string]any)
		switch ev.Event {
		case ai.EventMessage:
			c := str(data, "content")
			assistant.WriteString(c)
			fmt.Fprint(r.out, c)
		case ai.EventToolCall:
			fmt.Fprint(r.out, r.dim(fmt.Sprintf("\n  ⚙ %s %s\n", str(data, "tool"), compactArgs(data["args"]))))
		case ai.EventToolResult:
			if b, _ := data["is_error"].(bool); b {
				fmt.Fprint(r.out, r.dim(fmt.Sprintf("  ✗ %s\n", str(data, "result"))))
			}
		case ai.EventActionRequired:
			pending = str(data, "session_id")
			r.printProposedAction(data)
		case ai.EventError:
			fmt.Fprint(r.out, r.dim(fmt.Sprintf("\n  ⚠ %s\n", str(data, "message"))))
		}
	}

	r.svc.Chat(ctx, ai.ChatParams{ClusterID: 1, UserRole: "super-admin", History: r.history}, emit)

	// Resolve any mutation approvals; ContinueAction may queue another.
	for pending != "" {
		sid := pending
		pending = ""
		if r.confirm("  Apply this change? [y/N] ") {
			r.svc.ContinueAction(ctx, sid, ai.Actor{Role: "super-admin"}, emit)
		} else {
			fmt.Fprintln(r.out, r.dim("  cancelled."))
		}
	}

	fmt.Fprintln(r.out)
	r.history = append(r.history, ai.Message{Role: "assistant", Content: assistant.String()})
}

func (r *aiRunner) printProposedAction(data map[string]any) {
	fmt.Fprint(r.out, r.dim(fmt.Sprintf("\n  proposed: %s %s\n", str(data, "tool"), compactArgs(data["args"]))))
	if args, ok := data["args"].(map[string]any); ok {
		for _, k := range []string{"yaml", "patch"} {
			if v := str(args, k); v != "" {
				fmt.Fprintln(r.out, indent(v))
			}
		}
	}
}

func (r *aiRunner) confirm(prompt string) bool {
	fmt.Fprint(r.out, prompt)
	line, err := r.in.ReadString('\n')
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

func (r *aiRunner) dim(s string) string {
	if !r.color {
		return s
	}
	return "\033[2m" + s + "\033[0m"
}

// ---- helpers ----

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func resolveKubeconfig(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kube", "config")
}

func currentContext(kubeData []byte) string {
	cfg, err := clientcmd.Load(kubeData)
	if err != nil || cfg.CurrentContext == "" {
		return "default"
	}
	return cfg.CurrentContext
}

func str(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

func compactArgs(v any) string {
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return ""
	}
	// Show only short scalar fields; skip large blobs like yaml/patch.
	out := map[string]any{}
	for k, val := range m {
		if k == "yaml" || k == "patch" {
			continue
		}
		out[k] = val
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func indent(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "    " + l
	}
	return strings.Join(lines, "\n")
}

// ---- static repository adapters (DB-less CLI) ----

// staticSettingRepo serves a fixed AI config so ai.ConfigStore works without a
// database.
type staticSettingRepo struct{ cfg ai.Config }

func (s *staticSettingRepo) Get(context.Context, string) (*model.Setting, error) {
	b, err := json.Marshal(s.cfg)
	if err != nil {
		return nil, err
	}
	return &model.Setting{Key: "ai.config", Value: string(b), Category: "ai"}, nil
}
func (s *staticSettingRepo) Set(context.Context, *model.Setting) error             { return nil }
func (s *staticSettingRepo) List(context.Context, string) ([]model.Setting, error) { return nil, nil }
func (s *staticSettingRepo) Delete(context.Context, string) error                  { return nil }

// staticClusterRepo resolves the single local cluster (id 1) used by the CLI.
type staticClusterRepo struct{ name string }

func (s *staticClusterRepo) GetByID(_ context.Context, id uint) (*model.Cluster, error) {
	return &model.Cluster{ID: id, Name: s.name}, nil
}
func (s *staticClusterRepo) GetByName(_ context.Context, name string) (*model.Cluster, error) {
	return &model.Cluster{ID: 1, Name: name}, nil
}
func (s *staticClusterRepo) Create(context.Context, *model.Cluster) error  { return nil }
func (s *staticClusterRepo) Update(context.Context, *model.Cluster) error  { return nil }
func (s *staticClusterRepo) Delete(context.Context, uint) error            { return nil }
func (s *staticClusterRepo) List(context.Context) ([]model.Cluster, error) { return nil, nil }

// staticRoleRepo is never consulted because the CLI user runs as super-admin
// (their kubeconfig RBAC is the real authority), but the Service requires one.
type staticRoleRepo struct{}

func (s *staticRoleRepo) GetByName(context.Context, string) (*model.Role, error) { return nil, nil }
func (s *staticRoleRepo) GetByID(context.Context, uint) (*model.Role, error)     { return nil, nil }
func (s *staticRoleRepo) Create(context.Context, *model.Role) error              { return nil }
func (s *staticRoleRepo) Update(context.Context, *model.Role) error              { return nil }
func (s *staticRoleRepo) Delete(context.Context, uint) error                     { return nil }
func (s *staticRoleRepo) List(context.Context) ([]model.Role, error)             { return nil, nil }
