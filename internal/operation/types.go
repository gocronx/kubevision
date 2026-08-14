package operation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gocronx/kubevision/internal/model"
)

const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
)

type Input struct {
	UserID       uint
	Username     string
	Kind         string
	Action       string
	Cluster      string
	Namespace    string
	ResourceName string
	RequestID    string
	Payload      any
}

type Principal struct {
	UserID   uint
	Username string
	Role     string
}

type Reporter func(stage, message string, progress int)

type Executor func(context.Context, Principal, json.RawMessage, Reporter) (any, *Failure)

type Failure struct {
	Stage             string
	Code              string
	Message           string
	Suggestions       []string
	Retryable         bool
	RollbackAvailable bool
}

type View struct {
	ID                string                 `json:"id"`
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
	StartedAt         *time.Time             `json:"startedAt,omitempty"`
	CompletedAt       *time.Time             `json:"completedAt,omitempty"`
	ParentID          string                 `json:"parentId,omitempty"`
	UserID            uint                   `json:"userId"`
	Username          string                 `json:"username"`
	Kind              string                 `json:"kind"`
	Action            string                 `json:"action"`
	Status            string                 `json:"status"`
	Stage             string                 `json:"stage"`
	Cluster           string                 `json:"cluster,omitempty"`
	Namespace         string                 `json:"namespace,omitempty"`
	ResourceName      string                 `json:"resourceName,omitempty"`
	Progress          int                    `json:"progress"`
	ErrorCode         string                 `json:"errorCode,omitempty"`
	ErrorMessage      string                 `json:"errorMessage,omitempty"`
	Suggestions       []string               `json:"suggestions,omitempty"`
	RequestID         string                 `json:"requestId,omitempty"`
	Retryable         bool                   `json:"retryable"`
	RollbackAvailable bool                   `json:"rollbackAvailable"`
	Result            map[string]interface{} `json:"result,omitempty"`
	Events            []model.OperationEvent `json:"events,omitempty"`
}

func IsActive(status string) bool { return status == StatusQueued || status == StatusRunning }
