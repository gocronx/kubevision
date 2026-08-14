package packages

import (
	"context"
	"time"
)

const (
	PermissionRead     = "package-releases:read"
	PermissionInstall  = "package-releases:install"
	PermissionUpgrade  = "package-releases:upgrade"
	PermissionRollback = "package-releases:rollback"
	PermissionRemove   = "package-releases:remove"
)

type ChartSource struct {
	Chart               string `json:"chart,omitempty"`
	RepoURL             string `json:"repoUrl,omitempty"`
	Version             string `json:"version,omitempty"`
	RepositoryID        uint   `json:"repositoryId,omitempty"`
	UploadID            string `json:"uploadId,omitempty"`
	Username            string `json:"-"`
	Password            string `json:"-"`
	AllowPrivateNetwork bool   `json:"-"`
}

type ChangeOptions struct {
	ReleaseName       string                 `json:"releaseName" binding:"required"`
	Namespace         string                 `json:"namespace" binding:"required"`
	Source            ChartSource            `json:"source" binding:"required"`
	Values            map[string]interface{} `json:"values,omitempty"`
	CreateNamespace   bool                   `json:"createNamespace,omitempty"`
	Wait              bool                   `json:"wait,omitempty"`
	Atomic            bool                   `json:"atomic,omitempty"`
	Timeout           time.Duration          `json:"-"`
	ConfirmationToken string                 `json:"confirmationToken,omitempty"`
	ExpectedDigest    string                 `json:"-"`
}

type Risk struct {
	Level    string `json:"level"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Resource string `json:"resource,omitempty"`
}

type Preview struct {
	Operation         string        `json:"operation"`
	Chart             string        `json:"chart"`
	ChartVersion      string        `json:"chartVersion"`
	AppVersion        string        `json:"appVersion,omitempty"`
	Digest            string        `json:"digest"`
	Manifest          string        `json:"manifest"`
	Resources         []ResourceRef `json:"resources"`
	Risks             []Risk        `json:"risks"`
	CanExecute        bool          `json:"canExecute"`
	ConfirmationToken string        `json:"confirmationToken,omitempty"`
	ExpiresAt         time.Time     `json:"expiresAt,omitempty"`
}

type Actor struct {
	UserID      uint
	Username    string
	Role        string
	ClientIP    string
	PreviewOnly bool
}

type ResourceRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
}

type Release struct {
	Name         string                 `json:"name"`
	Namespace    string                 `json:"namespace"`
	Revision     int                    `json:"revision"`
	Status       string                 `json:"status"`
	Chart        string                 `json:"chart"`
	ChartVersion string                 `json:"chartVersion"`
	AppVersion   string                 `json:"appVersion"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	Notes        string                 `json:"notes,omitempty"`
	Values       map[string]interface{} `json:"values,omitempty"`
	Resources    []ResourceRef          `json:"resources,omitempty"`
}

type ListOptions struct {
	Namespace string
	State     string
	Label     string
	Limit     int
}

type RollbackOptions struct {
	Revision int           `json:"revision" binding:"required,min=1"`
	Wait     bool          `json:"wait"`
	Atomic   bool          `json:"atomic"`
	Timeout  time.Duration `json:"-"`
}

type RemoveOptions struct {
	Confirmation string        `json:"confirmation" binding:"required"`
	KeepHistory  bool          `json:"keepHistory"`
	Wait         bool          `json:"wait"`
	Timeout      time.Duration `json:"-"`
}

type Adapter interface {
	List(context.Context, string, ListOptions) ([]Release, error)
	Get(context.Context, string, string, string, int) (*Release, error)
	History(context.Context, string, string, string) ([]Release, error)
	Rollback(context.Context, string, string, string, RollbackOptions) error
	Remove(context.Context, string, string, string, RemoveOptions) error
	Preview(context.Context, string, string, ChangeOptions) (*Preview, error)
	Install(context.Context, string, ChangeOptions) error
	Upgrade(context.Context, string, ChangeOptions) error
}

type Authorizer interface {
	Allowed(context.Context, Actor, string, string, string) bool
}

type AuditEvent struct {
	Actor     Actor
	Action    string
	Cluster   string
	Namespace string
	Release   string
	Revision  int
	Options   string
	Outcome   string
	Duration  time.Duration
}

type Auditor interface{ RecordPackageAudit(AuditEvent) }
