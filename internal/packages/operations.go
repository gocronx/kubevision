package packages

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gocronx/kubevision/internal/operation"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
)

const OperationKind = "helm_release"

type OperationTask struct {
	Action          string                 `json:"action"`
	Cluster         string                 `json:"cluster"`
	Namespace       string                 `json:"namespace"`
	Name            string                 `json:"name"`
	Source          ChartSource            `json:"source,omitempty"`
	Values          map[string]interface{} `json:"values,omitempty"`
	CreateNamespace bool                   `json:"createNamespace,omitempty"`
	Wait            bool                   `json:"wait,omitempty"`
	Atomic          bool                   `json:"atomic,omitempty"`
	TimeoutSeconds  int                    `json:"timeoutSeconds,omitempty"`
	ExpectedDigest  string                 `json:"expectedDigest,omitempty"`
	Revision        int                    `json:"revision,omitempty"`
	Confirmation    string                 `json:"confirmation,omitempty"`
	KeepHistory     bool                   `json:"keepHistory,omitempty"`
}

func NewChangeOperationTask(prepared PreparedChange) OperationTask {
	return OperationTask{Action: prepared.Operation, Cluster: prepared.Cluster, Namespace: prepared.Options.Namespace,
		Name: prepared.Options.ReleaseName, Source: prepared.Options.Source, Values: prepared.Options.Values,
		CreateNamespace: prepared.Options.CreateNamespace, Wait: prepared.Options.Wait, Atomic: prepared.Options.Atomic,
		TimeoutSeconds: int(prepared.Options.Timeout / time.Second), ExpectedDigest: prepared.ExpectedDigest}
}

func NewOperationExecutor(service *Service) operation.Executor {
	return func(ctx context.Context, principal operation.Principal, raw json.RawMessage, report operation.Reporter) (any, *operation.Failure) {
		var task OperationTask
		if err := json.Unmarshal(raw, &task); err != nil {
			return nil, &operation.Failure{Stage: "preparing", Code: "INVALID_OPERATION_INPUT", Message: "The saved operation input is invalid"}
		}
		actor := Actor{UserID: principal.UserID, Username: principal.Username, Role: principal.Role}
		timeout := time.Duration(task.TimeoutSeconds) * time.Second
		report("applying_resources", "Applying Helm release resources", 35)
		var err error
		switch task.Action {
		case "install", "upgrade":
			err = service.ExecutePreparedChange(ctx, actor, PreparedChange{Operation: task.Action, Cluster: task.Cluster,
				Options: ChangeOptions{ReleaseName: task.Name, Namespace: task.Namespace, Source: task.Source, Values: task.Values,
					CreateNamespace: task.CreateNamespace, Wait: task.Wait, Atomic: task.Atomic, Timeout: timeout}, ExpectedDigest: task.ExpectedDigest})
		case "rollback":
			err = service.Rollback(ctx, actor, task.Cluster, task.Namespace, task.Name, RollbackOptions{Revision: task.Revision, Wait: task.Wait, Atomic: task.Atomic, Timeout: timeout})
		case "remove":
			err = service.Remove(ctx, actor, task.Cluster, task.Namespace, task.Name, RemoveOptions{Confirmation: task.Confirmation, KeepHistory: task.KeepHistory, Wait: task.Wait, Timeout: timeout})
		default:
			return nil, &operation.Failure{Stage: "preparing", Code: "UNSUPPORTED_OPERATION", Message: "This package operation is not supported"}
		}
		if err != nil {
			return nil, packageOperationFailure(err, task.Action)
		}
		report("verifying_release", "Verifying Helm release state", 90)
		if task.Action == "remove" {
			return map[string]interface{}{"removed": true}, nil
		}
		release, getErr := service.Get(ctx, actor, task.Cluster, task.Namespace, task.Name, 0)
		if getErr != nil {
			return map[string]interface{}{"releaseName": task.Name, "namespace": task.Namespace}, nil
		}
		return release, nil
	}
}

func packageOperationFailure(err error, action string) *operation.Failure {
	failure := &operation.Failure{Stage: "applying_resources", Code: "PACKAGE_OPERATION_FAILED", Message: err.Error(), RollbackAvailable: action == "upgrade"}
	if business, ok := err.(*bizerr.BizError); ok {
		failure.Message = business.Message
		switch business.Code {
		case bizerr.CodeForbidden:
			failure.Stage, failure.Code, failure.Retryable = "authorization", "PERMISSION_DENIED", false
		case bizerr.CodeValidation, bizerr.CodeParamInvalid:
			failure.Stage, failure.Code, failure.Retryable = "validation", "VALUES_OR_RESOURCE_INVALID", false
		case bizerr.CodeConflict:
			failure.Code, failure.Retryable = "RELEASE_CONFLICT", true
		case bizerr.CodeNotFound:
			failure.Code, failure.Retryable = "RELEASE_NOT_FOUND", false
		case bizerr.CodeK8sUnavailable:
			if strings.Contains(strings.ToLower(business.Message), "timed out") {
				failure.Stage, failure.Code = "wait_readiness", "POD_READINESS_TIMEOUT"
				failure.Suggestions = []string{"Inspect Pod status and events", "Inspect container logs", "Retry with a longer timeout"}
			} else {
				failure.Code = "KUBERNETES_UNAVAILABLE"
				failure.Suggestions = []string{"Check cluster connectivity", "Inspect Kubernetes events"}
			}
		}
	}
	return failure
}
