package ai

import (
	"context"
	"encoding/json"

	"github.com/gocronx/kubevision/internal/operation"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
)

const OperationKind = "ai_change"

type OperationTask struct {
	Action ApprovedAction `json:"action"`
}

func NewOperationExecutor(service *Service) operation.Executor {
	return func(ctx context.Context, principal operation.Principal, raw json.RawMessage, report operation.Reporter) (any, *operation.Failure) {
		var task OperationTask
		if err := json.Unmarshal(raw, &task); err != nil {
			return nil, &operation.Failure{Stage: "preparing", Code: "INVALID_OPERATION_INPUT", Message: "The saved AI action is invalid"}
		}
		report("applying_resources", "Executing approved AI change", 40)
		_, err := service.ExecuteApprovedAction(ctx, task.Action, Actor{UserID: principal.UserID, Username: principal.Username, Role: principal.Role, ClientIP: task.Action.ClientIP})
		if err != nil {
			failure := &operation.Failure{Stage: "applying_resources", Code: "AI_CHANGE_FAILED", Message: "The approved AI change failed",
				Suggestions: []string{"Inspect the target resources and Kubernetes events", "Ask the AI assistant to inspect the current state before proposing another change"}}
			if business, ok := err.(*bizerr.BizError); ok {
				failure.Message = business.Message
				if business.Code == bizerr.CodeForbidden {
					failure.Stage, failure.Code = "authorization", "PERMISSION_DENIED"
				} else if business.Code == bizerr.CodeValidation {
					failure.Stage, failure.Code = "validation", "AI_CONFIGURATION_INVALID"
				}
			}
			return nil, failure
		}
		report("verifying_release", "AI change completed", 90)
		return map[string]interface{}{"tool": task.Action.ToolCall.Function.Name, "applied": true}, nil
	}
}
