package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

type UpgradePolicyInput struct {
	Cluster           string                 `json:"cluster" binding:"required"`
	Namespace         string                 `json:"namespace" binding:"required"`
	ReleaseName       string                 `json:"releaseName" binding:"required"`
	RepositoryID      uint                   `json:"repositoryId" binding:"required"`
	Chart             string                 `json:"chart" binding:"required"`
	VersionConstraint string                 `json:"versionConstraint"`
	Values            map[string]interface{} `json:"values,omitempty"`
	IntervalMinutes   int                    `json:"intervalMinutes"`
	Enabled           bool                   `json:"enabled"`
}

type UpgradeManager struct {
	db      *gorm.DB
	catalog *Catalog
	service *Service
	logger  *zap.Logger
	mu      sync.Mutex
	cancel  context.CancelFunc
}

func NewUpgradeManager(db *gorm.DB, catalog *Catalog, service *Service, logger *zap.Logger) *UpgradeManager {
	return &UpgradeManager{db: db, catalog: catalog, service: service, logger: logger}
}

func (m *UpgradeManager) List(ctx context.Context, actor Actor) ([]model.HelmUpgradePolicy, error) {
	if !isAdmin(actor) {
		return nil, bizerr.ErrForbidden
	}
	var items []model.HelmUpgradePolicy
	err := m.db.WithContext(ctx).Order("release_name ASC").Find(&items).Error
	return items, err
}

func (m *UpgradeManager) Save(ctx context.Context, actor Actor, id uint, input UpgradePolicyInput) (*model.HelmUpgradePolicy, error) {
	if !isAdmin(actor) {
		return nil, bizerr.ErrForbidden
	}
	input.Cluster, input.Namespace, input.ReleaseName, input.Chart = strings.TrimSpace(input.Cluster), strings.TrimSpace(input.Namespace), strings.TrimSpace(input.ReleaseName), strings.TrimSpace(input.Chart)
	if input.Cluster == "" || input.Namespace == "" || input.ReleaseName == "" || input.Chart == "" || input.RepositoryID == 0 {
		return nil, bizerr.New(bizerr.CodeParamInvalid, "cluster, namespace, release, repository, and chart are required")
	}
	constraint := strings.TrimSpace(input.VersionConstraint)
	if constraint == "" {
		constraint = "~0"
	}
	if _, err := semver.NewConstraint(constraint); err != nil {
		return nil, bizerr.New(bizerr.CodeValidation, "invalid semantic version constraint")
	}
	if input.IntervalMinutes < 15 || input.IntervalMinutes > 10080 {
		return nil, bizerr.New(bizerr.CodeValidation, "interval must be between 15 and 10080 minutes")
	}
	var repository model.HelmRepository
	if err := m.db.WithContext(ctx).First(&repository, input.RepositoryID).Error; err != nil {
		return nil, err
	}
	if repository.Type != "helm" {
		return nil, bizerr.New(bizerr.CodeValidation, "automatic upgrades currently require a managed Helm repository")
	}
	valuesJSON, err := json.Marshal(input.Values)
	if err != nil || len(valuesJSON) > 512*1024 {
		return nil, bizerr.New(bizerr.CodeValidation, "values are invalid or too large")
	}
	var item model.HelmUpgradePolicy
	if id != 0 {
		if err := m.db.WithContext(ctx).First(&item, id).Error; err != nil {
			return nil, err
		}
	}
	next := time.Now().UTC()
	item.Cluster, item.Namespace, item.ReleaseName, item.RepositoryID, item.Chart = input.Cluster, input.Namespace, input.ReleaseName, input.RepositoryID, input.Chart
	item.VersionConstraint, item.ValuesJSON, item.IntervalMinutes, item.Enabled, item.NextCheckAt = constraint, string(valuesJSON), input.IntervalMinutes, input.Enabled, &next
	if item.Status == "" {
		item.Status = "idle"
	}
	if id == 0 {
		err = m.db.WithContext(ctx).Create(&item).Error
	} else {
		err = m.db.WithContext(ctx).Save(&item).Error
	}
	return &item, err
}

func (m *UpgradeManager) Delete(ctx context.Context, actor Actor, id uint) error {
	if !isAdmin(actor) {
		return bizerr.ErrForbidden
	}
	return m.db.WithContext(ctx).Delete(&model.HelmUpgradePolicy{}, id).Error
}

func (m *UpgradeManager) CheckNow(ctx context.Context, actor Actor, id uint) (*model.HelmUpgradePolicy, error) {
	if !isAdmin(actor) {
		return nil, bizerr.ErrForbidden
	}
	var item model.HelmUpgradePolicy
	if err := m.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	m.check(ctx, &item, actor)
	return &item, nil
}

func (m *UpgradeManager) Start(parent context.Context) {
	m.mu.Lock()
	if m.cancel != nil {
		m.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	m.cancel = cancel
	m.mu.Unlock()
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.runDue(ctx)
			}
		}
	}()
}

func (m *UpgradeManager) Stop() {
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.mu.Unlock()
}

func (m *UpgradeManager) runDue(ctx context.Context) {
	var items []model.HelmUpgradePolicy
	now := time.Now().UTC()
	if m.db.WithContext(ctx).Where("enabled = ? AND (next_check_at IS NULL OR next_check_at <= ?)", true, now).Limit(25).Find(&items).Error != nil {
		return
	}
	for i := range items {
		if ctx.Err() != nil {
			return
		}
		m.check(ctx, &items[i], Actor{Username: "helm-auto-upgrade", Role: "super-admin"})
	}
}

func (m *UpgradeManager) check(ctx context.Context, item *model.HelmUpgradePolicy, actor Actor) {
	now := time.Now().UTC()
	next := now.Add(time.Duration(item.IntervalMinutes) * time.Minute)
	updates := map[string]interface{}{"last_checked_at": &now, "next_check_at": &next}
	version, err := m.latestVersion(ctx, item)
	if err != nil {
		updates["status"], updates["last_error"] = "failed", err.Error()
		m.update(item, updates)
		return
	}
	release, err := m.service.Get(ctx, actor, item.Cluster, item.Namespace, item.ReleaseName, 0)
	if err != nil {
		updates["status"], updates["last_error"] = "failed", err.Error()
		m.update(item, updates)
		return
	}
	current, currentErr := semver.NewVersion(release.ChartVersion)
	target, targetErr := semver.NewVersion(version)
	if currentErr != nil || targetErr != nil || !target.GreaterThan(current) {
		updates["status"], updates["last_error"], updates["last_version"] = "current", "", version
		m.update(item, updates)
		return
	}
	var values map[string]interface{}
	if json.Unmarshal([]byte(item.ValuesJSON), &values) != nil {
		updates["status"], updates["last_error"] = "failed", "stored values are invalid"
		m.update(item, updates)
		return
	}
	opts := ChangeOptions{ReleaseName: item.ReleaseName, Namespace: item.Namespace, Source: ChartSource{RepositoryID: item.RepositoryID, Chart: item.Chart, Version: version}, Values: values, Wait: true, Atomic: true, Timeout: 10 * time.Minute}
	previewActor := actor
	previewActor.PreviewOnly = true
	preview, err := m.service.Preview(ctx, previewActor, "upgrade", item.Cluster, opts)
	if err != nil {
		updates["status"], updates["last_error"] = "blocked", err.Error()
		m.update(item, updates)
		return
	}
	if hasCriticalRisk(preview.Risks) {
		updates["status"], updates["last_error"], updates["last_version"] = "blocked", "new version requires manual review", version
		m.update(item, updates)
		return
	}
	approved, err := m.service.Preview(ctx, actor, "upgrade", item.Cluster, opts)
	if err != nil {
		updates["status"], updates["last_error"] = "blocked", err.Error()
		m.update(item, updates)
		return
	}
	opts.ConfirmationToken = approved.ConfirmationToken
	if err := m.service.Upgrade(ctx, actor, item.Cluster, opts); err != nil {
		updates["status"], updates["last_error"] = "failed", err.Error()
		m.update(item, updates)
		return
	}
	updates["status"], updates["last_error"], updates["last_version"] = "upgraded", "", version
	m.update(item, updates)
}

func (m *UpgradeManager) latestVersion(ctx context.Context, policy *model.HelmUpgradePolicy) (string, error) {
	source, repository, err := m.catalog.repositorySource(ctx, policy.RepositoryID)
	if err != nil {
		return "", err
	}
	if repository.Type != "helm" {
		return "", fmt.Errorf("automatic OCI upgrades are not supported")
	}
	data, err := fetchWithSource(ctx, strings.TrimRight(source.RepoURL, "/")+"/index.yaml", 10<<20, source)
	if err != nil {
		return "", err
	}
	var index repo.IndexFile
	if err := yaml.Unmarshal(data, &index); err != nil {
		return "", err
	}
	versions := index.Entries[policy.Chart]
	if len(versions) == 0 {
		return "", fmt.Errorf("chart not found")
	}
	constraint, _ := semver.NewConstraint(policy.VersionConstraint)
	candidates := []*semver.Version{}
	for _, item := range versions {
		version, err := semver.NewVersion(item.Version)
		if err == nil && constraint.Check(version) {
			candidates = append(candidates, version)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no version matches the constraint")
	}
	sort.Sort(sort.Reverse(semver.Collection(candidates)))
	return candidates[0].Original(), nil
}

func (m *UpgradeManager) update(item *model.HelmUpgradePolicy, updates map[string]interface{}) {
	if err := m.db.Model(item).Updates(updates).Error; err != nil && m.logger != nil {
		m.logger.Warn("failed to update Helm upgrade policy", zap.Uint("id", item.ID), zap.Error(err))
	}
	for key, value := range updates {
		switch key {
		case "status":
			item.Status = value.(string)
		case "last_error":
			item.LastError = value.(string)
		case "last_version":
			item.LastVersion = value.(string)
		}
	}
}
