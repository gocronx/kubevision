package packages

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

const (
	PermissionRepositoryRead        = "helm-repositories:read"
	PermissionRepositoryWrite       = "helm-repositories:write"
	maxUploadedChartBytes     int64 = 50 << 20
)

type RepositoryInput struct {
	Name                string `json:"name" binding:"required"`
	Type                string `json:"type" binding:"required"`
	URL                 string `json:"url" binding:"required"`
	Username            string `json:"username,omitempty"`
	Password            string `json:"password,omitempty"`
	AllowPrivateNetwork bool   `json:"allowPrivateNetwork"`
	Enabled             bool   `json:"enabled"`
}

type ChartSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version"`
	AppVersion  string   `json:"appVersion,omitempty"`
	Versions    []string `json:"versions,omitempty"`
}

type ChartInspection struct {
	ChartSummary
	Readme       string                 `json:"readme,omitempty"`
	Values       map[string]interface{} `json:"values"`
	Templates    []string               `json:"templates"`
	Dependencies []ChartSummary         `json:"dependencies,omitempty"`
	Digest       string                 `json:"digest"`
	UploadID     string                 `json:"uploadId,omitempty"`
}

type ArtifactPackage struct {
	PackageID     string `json:"packageId"`
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	Description   string `json:"description"`
	Version       string `json:"version"`
	AppVersion    string `json:"appVersion"`
	Repository    string `json:"repository"`
	RepositoryURL string `json:"repositoryUrl"`
}

type uploadEntry struct {
	data      []byte
	createdAt time.Time
	userID    uint
}

type Catalog struct {
	db             *gorm.DB
	encryptKey     string
	mu             sync.Mutex
	uploads        map[string]uploadEntry
	uploadSize     int64
	artifactClient *http.Client
}

func NewCatalog(db *gorm.DB, encryptKey string) *Catalog {
	return &Catalog{db: db, encryptKey: encryptKey, uploads: make(map[string]uploadEntry), artifactClient: publicHTTPClient()}
}

func (c *Catalog) SaveReleaseSource(ctx context.Context, cluster, namespace, releaseName string, source ChartSource) error {
	if source.UploadID != "" {
		return nil
	}
	item := model.HelmReleaseSource{
		Cluster:       cluster,
		Namespace:     namespace,
		ReleaseName:   releaseName,
		Chart:         strings.TrimSpace(source.Chart),
		RepositoryID:  source.RepositoryID,
		RepositoryURL: strings.TrimRight(strings.TrimSpace(source.RepoURL), "/"),
	}
	if item.Chart == "" {
		return bizerr.New(bizerr.CodeValidation, "chart source cannot be saved without a chart")
	}
	return c.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cluster"}, {Name: "namespace"}, {Name: "release_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"chart", "repository_id", "repository_url", "updated_at"}),
	}).Create(&item).Error
}

func (c *Catalog) ReleaseSource(ctx context.Context, cluster, namespace, releaseName string) (ChartSource, bool, error) {
	var item model.HelmReleaseSource
	err := c.db.WithContext(ctx).Where("cluster = ? AND namespace = ? AND release_name = ?", cluster, namespace, releaseName).First(&item).Error
	if err == gorm.ErrRecordNotFound {
		return ChartSource{}, false, nil
	}
	if err != nil {
		return ChartSource{}, false, err
	}
	return ChartSource{Chart: item.Chart, RepoURL: item.RepositoryURL, RepositoryID: item.RepositoryID}, true, nil
}

func (c *Catalog) DeleteReleaseSource(ctx context.Context, cluster, namespace, releaseName string) error {
	return c.db.WithContext(ctx).Where("cluster = ? AND namespace = ? AND release_name = ?", cluster, namespace, releaseName).Delete(&model.HelmReleaseSource{}).Error
}

func (c *Catalog) LatestVersion(ctx context.Context, actor Actor, source ChartSource) (ChartSummary, error) {
	if source.RepositoryID != 0 && !isAdmin(actor) {
		return ChartSummary{}, bizerr.ErrForbidden
	}
	resolved, err := c.ResolveSource(ctx, actor, source)
	if err != nil {
		return ChartSummary{}, err
	}
	if resolved.RepoURL == "" {
		return ChartSummary{}, bizerr.New(bizerr.CodeValidation, "one-click update checks require an indexed Helm repository")
	}
	data, err := fetchWithSource(ctx, strings.TrimRight(resolved.RepoURL, "/")+"/index.yaml", 10<<20, resolved)
	if err != nil {
		return ChartSummary{}, err
	}
	var index repo.IndexFile
	if err := yaml.Unmarshal(data, &index); err != nil {
		return ChartSummary{}, fmt.Errorf("parse repository index: %w", err)
	}
	return latestStableChartVersion(index, resolved.Chart)
}

func latestStableChartVersion(index repo.IndexFile, chartName string) (ChartSummary, error) {
	versions := index.Entries[chartName]
	if len(versions) == 0 {
		return ChartSummary{}, bizerr.New(bizerr.CodeNotFound, "chart was not found in the configured repository")
	}
	type candidate struct {
		version *semver.Version
		entry   *repo.ChartVersion
	}
	candidates := make([]candidate, 0, len(versions))
	for _, entry := range versions {
		version, parseErr := semver.NewVersion(entry.Version)
		if parseErr == nil && version.Prerelease() == "" {
			candidates = append(candidates, candidate{version: version, entry: entry})
		}
	}
	if len(candidates) == 0 {
		return ChartSummary{}, bizerr.New(bizerr.CodeValidation, "repository has no stable semantic chart versions")
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].version.GreaterThan(candidates[j].version) })
	latest := candidates[0].entry
	return ChartSummary{Name: chartName, Version: latest.Version, AppVersion: latest.AppVersion, Description: latest.Description}, nil
}

func (c *Catalog) ListRepositories(ctx context.Context, actor Actor) ([]model.HelmRepository, error) {
	if !isAdmin(actor) {
		return nil, bizerr.ErrForbidden
	}
	var items []model.HelmRepository
	if err := c.db.WithContext(ctx).Order("name ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		items[i].PasswordEnc = ""
	}
	return items, nil
}

func (c *Catalog) SaveRepository(ctx context.Context, actor Actor, id uint, input RepositoryInput) (*model.HelmRepository, error) {
	if !isAdmin(actor) {
		return nil, bizerr.ErrForbidden
	}
	input.Name, input.Type, input.URL = strings.TrimSpace(input.Name), strings.ToLower(strings.TrimSpace(input.Type)), strings.TrimRight(strings.TrimSpace(input.URL), "/")
	if input.Name == "" || input.URL == "" || (input.Type != "helm" && input.Type != "oci") {
		return nil, bizerr.New(bizerr.CodeParamInvalid, "name, URL, and a valid repository type are required")
	}
	if input.Type == "helm" {
		if err := validateRepositoryURL(input.URL, input.AllowPrivateNetwork); err != nil {
			return nil, bizerr.New(bizerr.CodeValidation, err.Error())
		}
	}
	if input.Type == "oci" {
		if err := validateOCIBase(input.URL, input.AllowPrivateNetwork); err != nil {
			return nil, bizerr.New(bizerr.CodeValidation, err.Error())
		}
	}
	var item model.HelmRepository
	if id != 0 {
		if err := c.db.WithContext(ctx).First(&item, id).Error; err != nil {
			return nil, err
		}
	}
	item.Name, item.Type, item.URL, item.Username = input.Name, input.Type, input.URL, strings.TrimSpace(input.Username)
	item.AllowPrivateNetwork, item.Enabled = input.AllowPrivateNetwork, input.Enabled
	if input.Password != "" {
		if c.encryptKey == "" {
			return nil, bizerr.New(bizerr.CodeValidation, "credential encryption is not configured")
		}
		encrypted, err := auth.Encrypt(input.Password, c.encryptKey)
		if err != nil {
			return nil, err
		}
		item.PasswordEnc = encrypted
	}
	if id == 0 {
		if err := c.db.WithContext(ctx).Create(&item).Error; err != nil {
			return nil, err
		}
	} else if err := c.db.WithContext(ctx).Save(&item).Error; err != nil {
		return nil, err
	}
	item.PasswordEnc = ""
	return &item, nil
}

func (c *Catalog) DeleteRepository(ctx context.Context, actor Actor, id uint) error {
	if !isAdmin(actor) {
		return bizerr.ErrForbidden
	}
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("repository_id = ?", id).Delete(&model.HelmUpgradePolicy{}).Error; err != nil {
			return err
		}
		if err := tx.Where("repository_id = ?", id).Delete(&model.HelmReleaseSource{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.HelmRepository{}, id).Error
	})
}

func (c *Catalog) TestRepository(ctx context.Context, actor Actor, id uint) error {
	if !isAdmin(actor) {
		return bizerr.ErrForbidden
	}
	source, repoModel, err := c.repositorySource(ctx, id)
	if err != nil {
		return err
	}
	if repoModel.Type == "helm" {
		_, err = fetchWithSource(ctx, strings.TrimRight(source.RepoURL, "/")+"/index.yaml", 10<<20, source)
	} else {
		err = validateChartSource(source)
	}
	now := time.Now().UTC()
	updates := map[string]interface{}{"last_checked_at": &now, "last_error": ""}
	if err != nil {
		updates["last_error"] = err.Error()
	}
	_ = c.db.WithContext(ctx).Model(&model.HelmRepository{}).Where("id = ?", id).Updates(updates).Error
	return err
}

func (c *Catalog) RepositoryCharts(ctx context.Context, actor Actor, id uint, query string) ([]ChartSummary, error) {
	if !isAdmin(actor) {
		return nil, bizerr.ErrForbidden
	}
	source, item, err := c.repositorySource(ctx, id)
	if err != nil {
		return nil, err
	}
	if item.Type != "helm" {
		return nil, bizerr.New(bizerr.CodeValidation, "OCI repositories do not provide a portable chart catalog; enter a chart path to inspect it")
	}
	data, err := fetchWithSource(ctx, strings.TrimRight(source.RepoURL, "/")+"/index.yaml", 10<<20, source)
	if err != nil {
		return nil, err
	}
	var index repo.IndexFile
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	out := make([]ChartSummary, 0, len(index.Entries))
	for name, versions := range index.Entries {
		if len(versions) == 0 || query != "" && !strings.Contains(strings.ToLower(name+" "+versions[0].Description), query) {
			continue
		}
		all := make([]string, 0, len(versions))
		for _, version := range versions {
			all = append(all, version.Version)
		}
		out = append(out, ChartSummary{Name: name, Description: versions[0].Description, Version: versions[0].Version, AppVersion: versions[0].AppVersion, Versions: all})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (c *Catalog) Inspect(ctx context.Context, actor Actor, source ChartSource) (*ChartInspection, error) {
	if source.RepositoryID != 0 && !isAdmin(actor) {
		return nil, bizerr.ErrForbidden
	}
	if err := c.AuthorizeUpload(actor, source.UploadID); err != nil {
		return nil, err
	}
	resolved, err := c.ResolveSource(ctx, actor, source)
	if err != nil {
		return nil, err
	}
	chrt, err := c.loadChart(ctx, resolved)
	if err != nil {
		return nil, err
	}
	return inspectChart(chrt), nil
}

func (c *Catalog) Upload(ctx context.Context, actor Actor, reader io.Reader, size int64) (*ChartInspection, error) {
	if !isAdmin(actor) && actor.Role != "editor" {
		return nil, bizerr.ErrForbidden
	}
	if size <= 0 || size > maxUploadedChartBytes {
		return nil, bizerr.New(bizerr.CodeParamInvalid, "chart archive must be between 1 byte and 50 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxUploadedChartBytes+1))
	if err != nil || int64(len(data)) > maxUploadedChartBytes {
		return nil, bizerr.New(bizerr.CodeParamInvalid, "chart archive is too large")
	}
	chrt, err := loader.LoadArchive(bytes.NewReader(data))
	if err != nil {
		return nil, bizerr.New(bizerr.CodeValidation, "invalid Helm chart archive")
	}
	if err := rejectClusterLookups(chrt); err != nil {
		return nil, bizerr.New(bizerr.CodeValidation, err.Error())
	}
	id, err := randomToken()
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.pruneUploadsLocked(time.Now())
	if len(c.uploads) >= 50 || c.uploadSize+int64(len(data)) > 256<<20 {
		c.mu.Unlock()
		return nil, bizerr.New(bizerr.CodeConflict, "temporary chart storage is full")
	}
	c.uploads[id] = uploadEntry{data: data, createdAt: time.Now(), userID: actor.UserID}
	c.uploadSize += int64(len(data))
	c.mu.Unlock()
	result := inspectChart(chrt)
	result.UploadID = id
	return result, nil
}

func (c *Catalog) ResolveSource(ctx context.Context, _ Actor, source ChartSource) (ChartSource, error) {
	if source.UploadID != "" {
		return source, nil
	}
	if source.RepositoryID == 0 {
		return source, nil
	}
	resolved, item, err := c.repositorySource(ctx, source.RepositoryID)
	if err != nil {
		return source, err
	}
	resolved.Chart, resolved.Version, resolved.RepositoryID = source.Chart, source.Version, source.RepositoryID
	if item.Type == "oci" {
		resolved.Chart = strings.TrimRight(item.URL, "/") + "/" + strings.TrimLeft(source.Chart, "/")
		resolved.RepoURL = ""
	}
	return resolved, nil
}

func (c *Catalog) AuthorizeUpload(actor Actor, uploadID string) error {
	if uploadID == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneUploadsLocked(time.Now())
	entry, ok := c.uploads[uploadID]
	if !ok {
		return bizerr.New(bizerr.CodeNotFound, "uploaded chart expired or was not found")
	}
	if entry.userID != actor.UserID && !isAdmin(actor) {
		return bizerr.ErrForbidden
	}
	return nil
}

func (c *Catalog) loadChart(ctx context.Context, source ChartSource) (*chart.Chart, error) {
	if source.UploadID == "" {
		return loadRemoteChart(ctx, source)
	}
	c.mu.Lock()
	c.pruneUploadsLocked(time.Now())
	entry, ok := c.uploads[source.UploadID]
	c.mu.Unlock()
	if !ok {
		return nil, bizerr.New(bizerr.CodeNotFound, "uploaded chart expired or was not found")
	}
	return loader.LoadArchive(bytes.NewReader(entry.data))
}

func (c *Catalog) repositorySource(ctx context.Context, id uint) (ChartSource, *model.HelmRepository, error) {
	var item model.HelmRepository
	if err := c.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return ChartSource{}, nil, err
	}
	if !item.Enabled {
		return ChartSource{}, nil, bizerr.New(bizerr.CodeValidation, "repository is disabled")
	}
	password := ""
	if item.PasswordEnc != "" {
		var err error
		password, err = auth.Decrypt(item.PasswordEnc, c.encryptKey)
		if err != nil {
			return ChartSource{}, nil, fmt.Errorf("decrypt repository credential: %w", err)
		}
	}
	return ChartSource{RepoURL: item.URL, Username: item.Username, Password: password, AllowPrivateNetwork: item.AllowPrivateNetwork, RepositoryID: item.ID}, &item, nil
}

func (c *Catalog) pruneUploadsLocked(now time.Time) {
	for id, entry := range c.uploads {
		if now.Sub(entry.createdAt) > 30*time.Minute {
			c.uploadSize -= int64(len(entry.data))
			delete(c.uploads, id)
		}
	}
}

func inspectChart(chrt *chart.Chart) *ChartInspection {
	result := &ChartInspection{ChartSummary: ChartSummary{Name: chrt.Name(), Version: chrt.Metadata.Version, AppVersion: chrt.Metadata.AppVersion, Description: chrt.Metadata.Description}, Values: chrt.Values, Templates: make([]string, 0, len(chrt.Templates)), Digest: chartDigest(chrt)}
	for _, file := range chrt.Raw {
		if strings.EqualFold(file.Name, "README.md") || strings.HasSuffix(strings.ToLower(file.Name), "/readme.md") {
			result.Readme = string(file.Data)
			break
		}
	}
	for _, template := range chrt.Templates {
		result.Templates = append(result.Templates, template.Name)
	}
	for _, dependency := range chrt.Dependencies() {
		result.Dependencies = append(result.Dependencies, ChartSummary{Name: dependency.Name(), Version: dependency.Metadata.Version, AppVersion: dependency.Metadata.AppVersion, Description: dependency.Metadata.Description})
	}
	sort.Strings(result.Templates)
	return result
}

func (c *Catalog) SearchArtifactHub(ctx context.Context, actor Actor, query string, limit int) ([]ArtifactPackage, error) {
	if strings.TrimSpace(query) == "" {
		return []ArtifactPackage{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	u := "https://artifacthub.io/api/v1/packages/search?kind=0&limit=" + fmt.Sprint(limit) + "&offset=0&ts_query_web=" + url.QueryEscape(query)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	response, err := c.artifactClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Artifact Hub returned status %d", response.StatusCode)
	}
	var payload struct {
		Packages []struct {
			PackageID   string `json:"package_id"`
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
			Description string `json:"description"`
			Version     string `json:"version"`
			AppVersion  string `json:"app_version"`
			Repository  struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"repository"`
		} `json:"packages"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 5<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]ArtifactPackage, 0, len(payload.Packages))
	for _, item := range payload.Packages {
		out = append(out, ArtifactPackage{PackageID: item.PackageID, Name: item.Name, DisplayName: item.DisplayName, Description: item.Description, Version: item.Version, AppVersion: item.AppVersion, Repository: item.Repository.Name, RepositoryURL: item.Repository.URL})
	}
	return out, nil
}

func isAdmin(actor Actor) bool { return actor.Role == "admin" || actor.Role == "super-admin" }
