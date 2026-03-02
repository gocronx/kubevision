package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户
type User struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	Username         string         `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Email            string         `gorm:"size:255" json:"email"`
	OAuthID          string         `gorm:"size:255" json:"-"`
	PasswordHash     string         `gorm:"size:255;not null" json:"-"`
	Role             string         `gorm:"size:32;default:viewer" json:"role"` // super-admin|admin|editor|viewer|custom
	AuthProvider     string         `gorm:"size:32;default:local" json:"authProvider"`
	TokenVersion     int            `gorm:"default:0" json:"-"`
	TOTPSecretEnc    string         `gorm:"size:512" json:"-"`
	TOTPEnabled      bool           `gorm:"default:false" json:"totpEnabled"`
	RecoveryCodesEnc string         `gorm:"size:1024" json:"-"`
	IsActive         bool           `gorm:"default:true" json:"isActive"`
	LastLoginAt      *time.Time     `json:"lastLoginAt"`
}

// Cluster 集群
type Cluster struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Name          string         `gorm:"uniqueIndex;size:64;not null" json:"name"`
	DisplayName   string         `gorm:"size:128" json:"displayName"`
	APIServer     string         `gorm:"size:512" json:"apiServer"`
	AuthType      string         `gorm:"size:32;default:kubeconfig" json:"authType"` // kubeconfig|token|in-cluster
	KubeconfigEnc string         `gorm:"type:text" json:"-"`
	TokenEnc      string         `gorm:"type:text" json:"-"`
	Status        string         `gorm:"size:32;default:unknown" json:"status"` // healthy|unhealthy|unknown
	Version       string         `gorm:"size:32" json:"version"`
}

// Role 角色
type Role struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Name        string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	DisplayName string    `gorm:"size:128" json:"displayName"`
	IsSystem    bool      `gorm:"default:false" json:"isSystem"`
	Permissions string    `gorm:"type:text" json:"permissions"` // JSON array
}

// UserClusterRole 用户-集群角色映射
type UserClusterRole struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	UserID     uint      `gorm:"index;not null" json:"userId"`
	ClusterID  uint      `gorm:"index;not null" json:"clusterId"`
	RoleID     uint      `gorm:"not null" json:"roleId"`
	Namespaces string    `gorm:"size:1024" json:"namespaces"` // 逗号分隔
}

// AuditLog 审计日志
type AuditLog struct {
	ID          uint      `gorm:"primarykey;autoIncrement" json:"id"`
	CreatedAt   time.Time `gorm:"index" json:"createdAt"`
	UserID      uint      `json:"userId"`
	Username    string    `gorm:"size:64" json:"username"`
	Action      string    `gorm:"size:32;index" json:"action"` // create|update|delete|exec
	Resource    string    `gorm:"size:64" json:"resource"`
	Name        string    `gorm:"size:256" json:"name"`
	Namespace   string    `gorm:"size:64" json:"namespace"`
	Cluster     string    `gorm:"size:64" json:"cluster"`
	StatusCode  int       `json:"statusCode"`
	DurationMs  int64     `json:"durationMs"`
	ClientIP    string    `gorm:"size:64" json:"clientIp"`
	RequestBody string    `gorm:"type:text" json:"-"` // ≤4KB
}

// APIKey API密钥
type APIKey struct {
	ID        uint       `gorm:"primarykey" json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UserID    uint       `gorm:"index;not null" json:"userId"`
	Name      string     `gorm:"size:128;not null" json:"name"`
	KeyHash   string     `gorm:"uniqueIndex;size:255;not null" json:"-"`
	KeyPrefix string     `gorm:"size:16" json:"keyPrefix"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

// Template 资源模板
type Template struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	Name         string    `gorm:"size:128;not null" json:"name"`
	Category     string    `gorm:"size:64" json:"category"`
	ResourceType string    `gorm:"size:64" json:"resourceType"`
	Content      string    `gorm:"type:text;not null" json:"content"` // YAML
	IsBuiltin    bool      `gorm:"default:false" json:"isBuiltin"`
}

// Setting 系统设置
type Setting struct {
	Key      string `gorm:"primarykey;size:128" json:"key"`
	Value    string `gorm:"type:text" json:"value"`
	Category string `gorm:"size:64" json:"category"`
}

// Webhook 通知
type Webhook struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	URL       string    `gorm:"size:512;not null" json:"url"`
	Secret    string    `gorm:"size:256" json:"-"`
	Events    string    `gorm:"type:text" json:"events"`    // JSON: ["delete","scale"]
	Clusters  string    `gorm:"type:text" json:"clusters"`  // JSON: ["prod"]
	Resources string    `gorm:"type:text" json:"resources"` // JSON: ["deployments"]
	IsActive  bool      `gorm:"default:true" json:"isActive"`
}

// Favorite 收藏夹
type Favorite struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	UserID       uint      `gorm:"index;not null" json:"userId"`
	ClusterID    string    `gorm:"size:64" json:"clusterId"`
	Namespace    string    `gorm:"size:64" json:"namespace"`
	ResourceType string    `gorm:"size:64" json:"resourceType"`
	ResourceName string    `gorm:"size:256" json:"resourceName"`
	DisplayName  string    `gorm:"size:256" json:"displayName"`
	SortOrder    int       `gorm:"default:0" json:"sortOrder"`
}

// TerminalSession 终端会话录制
type TerminalSession struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	UserID     uint      `gorm:"index;not null" json:"userId"`
	Cluster    string    `gorm:"size:64" json:"cluster"`
	Namespace  string    `gorm:"size:64" json:"namespace"`
	Pod        string    `gorm:"size:256" json:"pod"`
	Container  string    `gorm:"size:128" json:"container"`
	Recording  string    `gorm:"type:text" json:"-"` // asciinema v2
	DurationMs int64     `json:"durationMs"`
	ExpiresAt  time.Time `gorm:"index" json:"expiresAt"`
}

// PluginConfig 插件配置
type PluginConfig struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	Name       string         `gorm:"uniqueIndex;size:64;not null" json:"name"`
	PluginType string         `gorm:"size:32;not null" json:"pluginType"` // monitoring|gitops|dashboard
	ClusterID  string         `gorm:"size:64" json:"clusterId"`
	Enabled    bool           `gorm:"default:false" json:"enabled"`
	Config     string         `gorm:"type:text" json:"config"` // JSON
}
