package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
	"gorm.io/gorm"
)

type RoleAuthorizer struct {
	roles repository.RoleRepo
	db    *gorm.DB
}

func NewRoleAuthorizer(roles repository.RoleRepo, db *gorm.DB) *RoleAuthorizer {
	return &RoleAuthorizer{roles: roles, db: db}
}

func (a *RoleAuthorizer) Allowed(ctx context.Context, actor Actor, permission, cluster, namespace string) bool {
	if actor.Role == "super-admin" || actor.Role == "admin" {
		return true
	}
	roleName, scoped := a.scopedRole(ctx, actor.UserID, cluster, namespace)
	if scoped && roleName == "" {
		return false
	}
	if roleName == "" {
		roleName = actor.Role
	}
	role, err := a.roles.GetByName(ctx, roleName)
	if err != nil {
		return false
	}
	var permissions []string
	if json.Unmarshal([]byte(role.Permissions), &permissions) != nil {
		return false
	}
	for _, candidate := range permissions {
		if candidate == "*:*" || candidate == permission {
			return true
		}
		if candidate == "package-releases:*" {
			return true
		}
	}
	return false
}

func (a *RoleAuthorizer) scopedRole(ctx context.Context, userID uint, cluster, namespace string) (string, bool) {
	if a.db == nil || userID == 0 {
		return "", false
	}
	var count int64
	if a.db.WithContext(ctx).Model(&model.UserClusterRole{}).Where("user_id = ?", userID).Count(&count).Error != nil || count == 0 {
		return "", false
	}
	var row struct {
		RoleName   string
		Namespaces string
	}
	err := a.db.WithContext(ctx).Table("user_cluster_roles AS ucr").Select("roles.name AS role_name, ucr.namespaces").Joins("JOIN clusters ON clusters.id = ucr.cluster_id").Joins("JOIN roles ON roles.id = ucr.role_id").Where("ucr.user_id = ? AND clusters.name = ?", userID, cluster).Take(&row).Error
	if err != nil {
		return "", true
	}
	if strings.TrimSpace(row.Namespaces) == "" {
		return row.RoleName, true
	}
	if namespace == "" {
		return "", true
	}
	for _, allowed := range strings.Split(row.Namespaces, ",") {
		if strings.TrimSpace(allowed) == namespace {
			return row.RoleName, true
		}
	}
	return "", true
}

type AuditRecorder interface{ Record(model.AuditLog) }
type AuditBridge struct{ audit AuditRecorder }

func NewAuditBridge(audit AuditRecorder) *AuditBridge { return &AuditBridge{audit: audit} }

func (a *AuditBridge) RecordPackageAudit(event AuditEvent) {
	if a == nil || a.audit == nil {
		return
	}
	a.audit.Record(model.AuditLog{
		UserID: event.Actor.UserID, Username: event.Actor.Username, Action: event.Action,
		Resource: "package-releases", Name: event.Release, Namespace: event.Namespace,
		Cluster: event.Cluster, StatusCode: auditStatus(event.Outcome), DurationMs: event.Duration.Milliseconds(),
		ClientIP:    event.Actor.ClientIP,
		RequestBody: fmt.Sprintf(`{"revision":%d,"options":%q,"outcome":%q}`, event.Revision, event.Options, event.Outcome),
	})
}

func auditStatus(outcome string) int {
	if outcome == "succeeded" {
		return 0
	}
	return 50000
}
