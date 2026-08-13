package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gocronx/kubevision/internal/kubernetes/cluster"
	"github.com/gocronx/kubevision/internal/kubernetes/resource"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
)

// maxIterations bounds how many tool-calling rounds a single chat turn may run
// before the agent gives up, guarding against runaway loops.
const maxIterations = 12

const maxEmptyResponses = 2

// Service is the long-lived AI assistant. It is injected once at startup and
// spawns a short-lived run per chat request.
type Service struct {
	cfg         *ConfigStore
	k8s         repository.K8sResourceRepo
	clusterMgr  *cluster.Manager
	clusterRepo repository.ClusterRepo
	registry    *resource.Registry
	authz       *authorizer
	prom        promGetter
	sessions    *sessionStore
	audit       AuditRecorder
}

// AuditRecorder is the narrow audit dependency used by AI mutations.
type AuditRecorder interface {
	Record(model.AuditLog)
}

// NewService wires the assistant with its collaborators.
func NewService(
	cfg *ConfigStore,
	k8s repository.K8sResourceRepo,
	clusterMgr *cluster.Manager,
	clusterRepo repository.ClusterRepo,
	registry *resource.Registry,
	roles repository.RoleRepo,
	prom promGetter,
	audit AuditRecorder,
) *Service {
	return &Service{
		cfg:         cfg,
		k8s:         k8s,
		clusterMgr:  clusterMgr,
		clusterRepo: clusterRepo,
		registry:    registry,
		authz:       newAuthorizer(roles),
		prom:        prom,
		sessions:    newSessionStore(),
		audit:       audit,
	}
}

// Config exposes the persisted configuration (used by the settings endpoints).
func (s *Service) Config() *ConfigStore { return s.cfg }

// ListModels discovers model IDs using an explicit configuration without
// persisting it. The settings handler uses this for its model picker.
func (s *Service) ListModels(ctx context.Context, cfg Config) ([]Model, error) {
	return NewClient(cfg).ListModels(ctx)
}

// ChatParams carries everything a single chat turn needs.
type ChatParams struct {
	ClusterID    uint
	UserID       uint
	Username     string
	UserRole     string
	ClientIP     string
	History      []Message // prior user/assistant turns (content only)
	Page         string
	Namespace    string
	ResourceKind string
	ResourceName string
}

// Chat runs a fresh chat turn, emitting events as the agent thinks, calls tools,
// and (for mutations) pauses for approval.
func (s *Service) Chat(ctx context.Context, p ChatParams, emit EmitFunc) {
	cfg, err := s.cfg.Load(ctx)
	if err != nil {
		emit(errorEvent("failed to load AI configuration"))
		return
	}
	if !cfg.Ready() {
		emit(errorEvent("AI assistant is not configured. Set an API key in settings."))
		return
	}

	cl, err := s.clusterRepo.GetByID(ctx, p.ClusterID)
	if err != nil || cl == nil {
		emit(errorEvent("cluster not found"))
		return
	}

	r := s.newRun(cfg, cl.Name, p.ClusterID, Actor{
		UserID: p.UserID, Username: p.Username, Role: p.UserRole, ClientIP: p.ClientIP,
	}, emit)
	system := buildSystemPrompt(promptContext{
		clusterName:  cl.Name,
		userRole:     p.UserRole,
		page:         p.Page,
		namespace:    p.Namespace,
		resourceKind: p.ResourceKind,
		resourceName: p.ResourceName,
	})
	messages := append([]Message{{Role: "system", Content: system}}, p.History...)
	r.loop(ctx, messages)
}

// ContinueAction resumes a paused conversation after the user approves a
// mutation identified by sessionID.
type Actor struct {
	UserID   uint
	Username string
	Role     string
	ClientIP string
}

func (s *Service) ContinueAction(ctx context.Context, sessionID string, actor Actor, emit EmitFunc) {
	sess, takeResult := s.sessions.takeOwned(sessionID, actor.UserID)
	if takeResult != sessionTaken {
		if sess != nil {
			s.recordApprovalAudit(sess, actor, takeResult)
		}
		if takeResult == sessionForbidden {
			emit(errorEvent("this action belongs to another user"))
			return
		}
		emit(errorEvent("this action has expired or was already handled"))
		return
	}

	cfg, err := s.cfg.Load(ctx)
	if err != nil || !cfg.Ready() {
		emit(errorEvent("AI assistant is not configured"))
		return
	}

	r := s.newRun(cfg, sess.clusterName, sess.clusterID, actor, emit)
	tc := sess.toolCall
	args := decodeArgs(tc.Function.Arguments)
	started := time.Now()

	// Re-authorize at execution time as defense in depth.
	var result string
	var isErr bool
	statusCode := http.StatusOK
	outcome := "succeeded"
	if deny := r.authz.authorize(ctx, actor.Role, tc.Function.Name, args); deny != "" {
		result, isErr = "Forbidden: "+deny, true
		statusCode, outcome = http.StatusForbidden, "denied"
	} else if res, execErr := r.exec.execute(ctx, tc.Function.Name, args); execErr != nil {
		result, isErr = "Tool error: "+execErr.Error(), true
		statusCode, outcome = http.StatusInternalServerError, "failed"
	} else {
		result = res
	}
	result = limitToolResult(result)
	s.recordMutationAudit(sess, actor, args, statusCode, outcome, time.Since(started))
	emit(toolResultEvent(tc.Function.Name, tc.ID, result, isErr))

	messages := append(sess.messages, toolResultMessage(tc.ID, result))
	r.loop(ctx, messages)
}

// llm is the chat-completions capability the agent loop depends on. *Client is
// the production implementation; tests substitute a scripted fake.
type llm interface {
	Stream(ctx context.Context, messages []Message, tools []Tool, onContent func(string)) (Message, error)
}

// run holds the per-request state for one chat turn.
type run struct {
	client      llm
	exec        *executor
	authz       *authorizer
	sessions    *sessionStore
	role        string
	userID      uint
	username    string
	clientIP    string
	clusterID   uint
	clusterName string
	emit        EmitFunc
}

func (s *Service) newRun(cfg Config, clusterName string, clusterID uint, actor Actor, emit EmitFunc) *run {
	return &run{
		client: NewClient(cfg),
		exec: &executor{
			k8s:         s.k8s,
			clusterMgr:  s.clusterMgr,
			registry:    s.registry,
			prom:        s.prom,
			clusterName: clusterName,
		},
		authz:       s.authz,
		sessions:    s.sessions,
		role:        actor.Role,
		userID:      actor.UserID,
		username:    actor.Username,
		clientIP:    actor.ClientIP,
		clusterID:   clusterID,
		clusterName: clusterName,
		emit:        emit,
	}
}

// loop drives the LLM/tool conversation until the model produces a final answer,
// pauses for a mutation approval, or hits the iteration ceiling.
func (r *run) loop(ctx context.Context, messages []Message) {
	tools := toolDefinitions()
	emptyResponses := 0

	for range maxIterations {
		assistant, err := r.client.Stream(ctx, messages, tools, func(delta string) {
			r.emit(messageEvent(delta))
		})
		if err != nil {
			r.emit(errorEvent(err.Error()))
			return
		}
		messages = append(messages, assistant)

		if len(assistant.ToolCalls) == 0 && strings.TrimSpace(assistant.Content) == "" {
			emptyResponses++
			if emptyResponses >= maxEmptyResponses {
				r.emit(errorEvent("the AI provider ended without a final response; please retry"))
				return
			}
			messages = append(messages, Message{
				Role:    "system",
				Content: "Your previous response was empty. Continue from the tool results and provide a concise final answer. If the requested change is ready, call the required mutation tool for approval.",
			})
			continue
		}
		emptyResponses = 0

		if len(assistant.ToolCalls) == 0 {
			r.emit(doneEvent())
			return
		}

		var pending *ToolCall
		var pendingArgs map[string]any

		for idx := range assistant.ToolCalls {
			tc := assistant.ToolCalls[idx]
			args := decodeArgs(tc.Function.Arguments)
			r.emit(toolCallEvent(tc.Function.Name, tc.ID, args))

			// Once a mutation is queued for approval, defer the rest of the turn.
			if pending != nil {
				const res = "Skipped: only one change is processed at a time. Re-request after the pending action is resolved."
				r.emit(toolResultEvent(tc.Function.Name, tc.ID, res, false))
				messages = append(messages, toolResultMessage(tc.ID, res))
				continue
			}

			if deny := r.authz.authorize(ctx, r.role, tc.Function.Name, args); deny != "" {
				res := "Forbidden: " + deny
				r.emit(toolResultEvent(tc.Function.Name, tc.ID, res, true))
				messages = append(messages, toolResultMessage(tc.ID, res))
				continue
			}

			if isMutation(tc.Function.Name) {
				m := tc
				pending = &m
				pendingArgs = args
				continue // result filled in on approval
			}

			res, execErr := r.exec.execute(ctx, tc.Function.Name, args)
			isErr := execErr != nil
			if isErr {
				res = "Tool error: " + execErr.Error()
			}
			res = limitToolResult(res)
			r.emit(toolResultEvent(tc.Function.Name, tc.ID, res, isErr))
			messages = append(messages, toolResultMessage(tc.ID, res))
		}

		if pending != nil {
			sid, ok := r.sessions.save(&pendingSession{
				messages:    messages,
				toolCall:    *pending,
				clusterID:   r.clusterID,
				clusterName: r.clusterName,
				userID:      r.userID,
				username:    r.username,
				clientIP:    r.clientIP,
			})
			if !ok {
				r.emit(errorEvent("too many pending AI actions; approve or dismiss an existing action before creating another"))
				return
			}
			r.emit(actionRequiredEvent(pending.Function.Name, pending.ID, pendingArgs, sid))
			return // wait for approval; the resume continues the loop
		}
	}

	r.emit(errorEvent("reached the maximum number of tool iterations"))
}

func (s *Service) recordApprovalAudit(sess *pendingSession, actor Actor, result sessionTakeResult) {
	if s.audit == nil {
		return
	}
	outcome := "denied"
	status := http.StatusForbidden
	if result == sessionExpired {
		outcome, status = "expired", http.StatusGone
	}
	s.audit.Record(model.AuditLog{
		CreatedAt: time.Now().UTC(), UserID: actor.UserID, Username: actor.Username,
		Action: "approve", Resource: "ai-action", Cluster: sess.clusterName,
		StatusCode: status, ClientIP: actor.ClientIP, Source: "ai-assistant",
		Tool: sess.toolCall.Function.Name, CorrelationID: sess.correlationID, Outcome: outcome,
	})
}

func (s *Service) recordMutationAudit(sess *pendingSession, actor Actor, args map[string]any, status int, outcome string, duration time.Duration) {
	if s.audit == nil {
		return
	}
	resource, name, namespace := mutationTarget(sess.toolCall.Function.Name, args)
	s.audit.Record(model.AuditLog{
		CreatedAt: time.Now().UTC(), UserID: actor.UserID, Username: actor.Username,
		Action: mutationAction(sess.toolCall.Function.Name), Resource: resource, Name: name,
		Namespace: namespace, Cluster: sess.clusterName, StatusCode: status,
		DurationMs: duration.Milliseconds(), ClientIP: actor.ClientIP, Source: "ai-assistant",
		Tool: sess.toolCall.Function.Name, CorrelationID: sess.correlationID, Outcome: outcome,
	})
}

func mutationAction(tool string) string {
	switch tool {
	case "create_resource":
		return "create"
	case "delete_resource":
		return "delete"
	default:
		return "update"
	}
}

func mutationTarget(tool string, args map[string]any) (resource, name, namespace string) {
	resource = argString(args, "kind")
	name = argString(args, "name")
	namespace = argString(args, "namespace")
	if tool == "create_resource" || tool == "update_resource" {
		if obj, err := parseManifest(argString(args, "yaml")); err == nil {
			if namespace == "" {
				namespace = manifestNamespace(obj)
			}
			if name == "" {
				if meta, ok := obj["metadata"].(map[string]any); ok {
					name, _ = meta["name"].(string)
				}
			}
		}
	}
	return
}

// decodeArgs parses a tool call's raw JSON arguments into a map, tolerating an
// empty argument string.
func decodeArgs(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return map[string]any{}
	}
	return args
}

// toolResultMessage builds the tool-role message that reports a tool's output
// back to the model.
func toolResultMessage(callID, content string) Message {
	return Message{Role: "tool", ToolCallID: callID, Content: content}
}
