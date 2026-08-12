package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/gocronx/kubevision/internal/auth"
	directoryclient "github.com/gocronx/kubevision/internal/directory"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/repository"
	"gorm.io/gorm"
)

type DirectorySettings struct {
	Enabled              bool                         `json:"enabled"`
	StartTLS             bool                         `json:"startTls"`
	AllowPlaintext       bool                         `json:"allowPlaintext"`
	RefreshMapping       bool                         `json:"refreshMapping"`
	ServerURL            string                       `json:"serverUrl"`
	CABundle             string                       `json:"caBundle"`
	BindDN               string                       `json:"bindDn"`
	BindPassword         string                       `json:"-"`
	UserBaseDN           string                       `json:"userBaseDn"`
	UserFilter           string                       `json:"userFilter"`
	StableIDAttribute    string                       `json:"stableIdAttribute"`
	UsernameAttribute    string                       `json:"usernameAttribute"`
	DisplayAttribute     string                       `json:"displayAttribute"`
	EmailAttribute       string                       `json:"emailAttribute"`
	GroupAttribute       string                       `json:"groupAttribute"`
	FallbackRole         string                       `json:"fallbackRole"`
	ConnectTimeoutSecs   int                          `json:"connectTimeoutSecs"`
	SearchTimeoutSecs    int                          `json:"searchTimeoutSecs"`
	Mappings             []model.DirectoryRoleMapping `json:"mappings"`
	CredentialConfigured bool                         `json:"credentialConfigured"`
}

type DirectoryPreview struct {
	Username    string                      `json:"username"`
	DisplayName string                      `json:"displayName"`
	Email       string                      `json:"email"`
	Groups      []string                    `json:"groups"`
	MatchedRule *model.DirectoryRoleMapping `json:"matchedRule,omitempty"`
	Role        string                      `json:"role"`
}

type DirectoryService struct {
	repo       repository.DirectoryRepo
	users      repository.DirectoryUserRepo
	roles      repository.RoleRepo
	client     directoryclient.Client
	encryptKey string
}

func NewDirectoryService(repo repository.DirectoryRepo, users repository.DirectoryUserRepo, roles repository.RoleRepo, client directoryclient.Client, encryptKey string) *DirectoryService {
	return &DirectoryService{repo: repo, users: users, roles: roles, client: client, encryptKey: encryptKey}
}

func defaultDirectorySettings() *DirectorySettings {
	return &DirectorySettings{StartTLS: true, ConnectTimeoutSecs: 5, SearchTimeoutSecs: 8, UserFilter: "(uid={{username}})", StableIDAttribute: "entryUUID", UsernameAttribute: "uid", DisplayAttribute: "displayName", EmailAttribute: "mail", GroupAttribute: "memberOf", FallbackRole: "viewer", Mappings: []model.DirectoryRoleMapping{}}
}

func (s *DirectoryService) GetSettings(ctx context.Context) (*DirectorySettings, error) {
	cfg, err := s.repo.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return defaultDirectorySettings(), nil
	}
	mappings, err := s.repo.ListMappings(ctx)
	if err != nil {
		return nil, err
	}
	return settingsFromModel(cfg, mappings), nil
}

func settingsFromModel(cfg *model.DirectoryConfig, mappings []model.DirectoryRoleMapping) *DirectorySettings {
	if mappings == nil {
		mappings = []model.DirectoryRoleMapping{}
	}
	return &DirectorySettings{Enabled: cfg.Enabled, StartTLS: cfg.StartTLS, AllowPlaintext: cfg.AllowPlaintext, RefreshMapping: cfg.RefreshMapping, ServerURL: cfg.ServerURL, CABundle: cfg.CABundle, BindDN: cfg.BindDN, UserBaseDN: cfg.UserBaseDN, UserFilter: cfg.UserFilter, StableIDAttribute: cfg.StableIDAttribute, UsernameAttribute: cfg.UsernameAttribute, DisplayAttribute: cfg.DisplayAttribute, EmailAttribute: cfg.EmailAttribute, GroupAttribute: cfg.GroupAttribute, FallbackRole: cfg.FallbackRole, ConnectTimeoutSecs: cfg.ConnectTimeoutSecs, SearchTimeoutSecs: cfg.SearchTimeoutSecs, Mappings: mappings, CredentialConfigured: cfg.BindPasswordEnc != ""}
}

func (s *DirectoryService) SaveSettings(ctx context.Context, in DirectorySettings) error {
	if err := s.validateSettings(ctx, in); err != nil {
		return err
	}
	current, err := s.repo.GetConfig(ctx)
	if err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to read directory settings")
	}
	passwordEnc := ""
	if current != nil {
		passwordEnc = current.BindPasswordEnc
	}
	if in.BindPassword != "" {
		passwordEnc, err = auth.Encrypt(in.BindPassword, s.encryptKey)
		if err != nil {
			return bizerr.New(bizerr.CodeInternal, "failed to protect bind credential")
		}
	}
	cfg := &model.DirectoryConfig{Enabled: in.Enabled, ServerURL: strings.TrimSpace(in.ServerURL), StartTLS: in.StartTLS, AllowPlaintext: in.AllowPlaintext, CABundle: in.CABundle, BindDN: strings.TrimSpace(in.BindDN), BindPasswordEnc: passwordEnc, ConnectTimeoutSecs: in.ConnectTimeoutSecs, SearchTimeoutSecs: in.SearchTimeoutSecs, UserBaseDN: strings.TrimSpace(in.UserBaseDN), UserFilter: strings.TrimSpace(in.UserFilter), StableIDAttribute: in.StableIDAttribute, UsernameAttribute: in.UsernameAttribute, DisplayAttribute: in.DisplayAttribute, EmailAttribute: in.EmailAttribute, GroupAttribute: in.GroupAttribute, FallbackRole: in.FallbackRole, RefreshMapping: in.RefreshMapping}
	sort.SliceStable(in.Mappings, func(i, j int) bool { return in.Mappings[i].Priority < in.Mappings[j].Priority })
	if err := s.repo.SaveConfig(ctx, cfg, in.Mappings); err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to save directory settings")
	}
	// A policy edit may remove privilege; revoke all directory sessions rather than
	// attempting to infer every user's current verified group set from stale data.
	users, err := s.users.ListByAuthProvider(ctx, "directory")
	if err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to revoke directory sessions")
	}
	for i := range users {
		users[i].TokenVersion++
		if err := s.users.Update(ctx, &users[i]); err != nil {
			return bizerr.New(bizerr.CodeInternal, "failed to revoke directory sessions")
		}
	}
	return nil
}

func (s *DirectoryService) validateSettings(ctx context.Context, in DirectorySettings) error {
	if in.ConnectTimeoutSecs < 1 || in.ConnectTimeoutSecs > 30 || in.SearchTimeoutSecs < 1 || in.SearchTimeoutSecs > 60 {
		return bizerr.New(bizerr.CodeParamInvalid, "directory timeouts are outside the allowed range")
	}
	if in.FallbackRole == "super-admin" {
		return bizerr.New(bizerr.CodeParamInvalid, "directory fallback cannot grant super-admin")
	}
	if _, err := s.roles.GetByName(ctx, in.FallbackRole); err != nil {
		return bizerr.New(bizerr.CodeParamInvalid, "fallback role does not exist")
	}
	seenGroup, seenPriority := map[string]bool{}, map[int]bool{}
	for _, m := range in.Mappings {
		if strings.TrimSpace(m.GroupID) == "" || seenGroup[m.GroupID] || seenPriority[m.Priority] || m.Role == "super-admin" {
			return bizerr.New(bizerr.CodeParamInvalid, "directory mappings must have unique exact groups and priorities")
		}
		if _, err := s.roles.GetByName(ctx, m.Role); err != nil {
			return bizerr.New(bizerr.CodeParamInvalid, "mapping role does not exist")
		}
		seenGroup[m.GroupID], seenPriority[m.Priority] = true, true
	}
	if !in.Enabled && strings.TrimSpace(in.ServerURL) == "" {
		return nil
	}
	if err := directoryclient.Validate(toClientConfig(in, "")); err != nil {
		return bizerr.New(bizerr.CodeParamInvalid, "invalid directory connection settings")
	}
	return nil
}

func (s *DirectoryService) Authenticate(ctx context.Context, identifier, password string) (*model.User, error) {
	settings, secret, err := s.runtimeSettings(ctx)
	if err != nil || !settings.Enabled {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "invalid username or password")
	}
	id, err := s.client.Authenticate(ctx, toClientConfig(*settings, secret), identifier, password)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeUnauthorized, "invalid username or password")
	}
	return s.reconcile(ctx, id, settings.Mappings, settings.FallbackRole)
}

func (s *DirectoryService) RefreshUser(ctx context.Context, user *model.User) error {
	settings, secret, err := s.runtimeSettings(ctx)
	if err != nil || !settings.Enabled || !settings.RefreshMapping {
		return err
	}
	id, err := s.client.Lookup(ctx, toClientConfig(*settings, secret), user.Username)
	if err != nil || id.StableID != user.DirectoryID {
		user.TokenVersion++
		_ = s.users.Update(ctx, user)
		return bizerr.New(bizerr.CodeUnauthorized, "directory identity could not be refreshed")
	}
	updated, err := s.reconcile(ctx, id, settings.Mappings, settings.FallbackRole)
	if err == nil {
		*user = *updated
	}
	return err
}

func (s *DirectoryService) Preview(ctx context.Context, identifier string) (*DirectoryPreview, error) {
	settings, secret, err := s.runtimeSettings(ctx)
	if err != nil {
		return nil, err
	}
	id, err := s.client.Lookup(ctx, toClientConfig(*settings, secret), identifier)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeK8sUnavailable, "directory lookup failed")
	}
	role, rule := selectRole(id.Groups, settings.Mappings, settings.FallbackRole)
	return &DirectoryPreview{Username: id.Username, DisplayName: id.DisplayName, Email: id.Email, Groups: id.Groups, MatchedRule: rule, Role: role}, nil
}

func (s *DirectoryService) TestConnection(ctx context.Context, candidate *DirectorySettings) string {
	settings, secret, err := s.runtimeSettings(ctx)
	if candidate != nil {
		settings = candidate
		err = nil
		if candidate.BindPassword != "" {
			secret = candidate.BindPassword
		}
	}
	if err != nil {
		return "configuration"
	}
	err = s.client.Ping(ctx, toClientConfig(*settings, secret))
	return directoryclient.Classify(err)
}

func (s *DirectoryService) runtimeSettings(ctx context.Context) (*DirectorySettings, string, error) {
	cfg, err := s.repo.GetConfig(ctx)
	if err != nil || cfg == nil {
		return nil, "", errors.New("directory not configured")
	}
	mappings, err := s.repo.ListMappings(ctx)
	if err != nil {
		return nil, "", err
	}
	secret := ""
	if cfg.BindPasswordEnc != "" {
		secret, err = auth.Decrypt(cfg.BindPasswordEnc, s.encryptKey)
		if err != nil {
			return nil, "", err
		}
	}
	return settingsFromModel(cfg, mappings), secret, nil
}

func (s *DirectoryService) reconcile(ctx context.Context, id *directoryclient.Identity, mappings []model.DirectoryRoleMapping, fallback string) (*model.User, error) {
	role, _ := selectRole(id.Groups, mappings, fallback)
	user, err := s.users.GetByDirectoryID(ctx, id.StableID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to reconcile directory identity")
	}
	if user == nil {
		// Username and email collisions require an explicit administrator link flow.
		if _, e := s.users.GetByUsername(ctx, id.Username); e == nil {
			return nil, bizerr.New(bizerr.CodeConflict, "directory identity requires administrator linking")
		}
		if id.Email != "" {
			if _, e := s.users.GetByEmail(ctx, id.Email); e == nil {
				return nil, bizerr.New(bizerr.CodeConflict, "directory identity requires administrator linking")
			}
		}
		user = &model.User{Username: id.Username, Email: id.Email, DisplayName: id.DisplayName, DirectoryID: id.StableID, AuthProvider: "directory", Role: role, PasswordHash: "", IsActive: true}
		if err := s.users.Create(ctx, user); err != nil {
			return nil, bizerr.New(bizerr.CodeConflict, "directory identity requires administrator linking")
		}
	} else {
		if !user.IsActive {
			return nil, bizerr.New(bizerr.CodeForbidden, "account is disabled")
		}
		if user.Role != role {
			user.Role = role
			user.TokenVersion++
		}
		user.Username, user.Email, user.DisplayName = id.Username, id.Email, id.DisplayName
	}
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.users.Update(ctx, user); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to update directory identity")
	}
	return user, nil
}

func selectRole(groups []string, mappings []model.DirectoryRoleMapping, fallback string) (string, *model.DirectoryRoleMapping) {
	set := make(map[string]struct{}, len(groups))
	for _, g := range groups {
		set[g] = struct{}{}
	}
	sort.SliceStable(mappings, func(i, j int) bool { return mappings[i].Priority < mappings[j].Priority })
	for i := range mappings {
		if _, ok := set[mappings[i].GroupID]; ok {
			m := mappings[i]
			return m.Role, &m
		}
	}
	return fallback, nil
}

func toClientConfig(in DirectorySettings, password string) directoryclient.Config {
	return directoryclient.Config{ServerURL: in.ServerURL, CABundle: in.CABundle, BindDN: in.BindDN, BindPassword: password, UserBaseDN: in.UserBaseDN, UserFilter: in.UserFilter, StableIDAttribute: in.StableIDAttribute, UsernameAttribute: in.UsernameAttribute, DisplayAttribute: in.DisplayAttribute, EmailAttribute: in.EmailAttribute, GroupAttribute: in.GroupAttribute, StartTLS: in.StartTLS, AllowPlaintext: in.AllowPlaintext, ConnectTimeout: time.Duration(in.ConnectTimeoutSecs) * time.Second, SearchTimeout: time.Duration(in.SearchTimeoutSecs) * time.Second}
}
