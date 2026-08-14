package packages

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
)

type Service struct {
	adapter  Adapter
	auth     Authorizer
	audit    Auditor
	mu       sync.Mutex
	active   map[string]struct{}
	previews map[string]previewGrant
	catalog  *Catalog
}

func (s *Service) WithCatalog(catalog *Catalog) *Service { s.catalog = catalog; return s }

type previewGrant struct {
	ActorID                                 uint
	Operation, Cluster, Fingerprint, Digest string
	ExpiresAt                               time.Time
}

func NewService(adapter Adapter, auth Authorizer, audit Auditor) *Service {
	return &Service{adapter: adapter, auth: auth, audit: audit, active: make(map[string]struct{}), previews: make(map[string]previewGrant)}
}

func (s *Service) Preview(ctx context.Context, actor Actor, operation, cluster string, opts ChangeOptions) (*Preview, error) {
	managedRepository := opts.Source.RepositoryID != 0
	if managedRepository && !isAdmin(actor) {
		return nil, bizerr.ErrForbidden
	}
	if s.catalog != nil {
		if uploadErr := s.catalog.AuthorizeUpload(actor, opts.Source.UploadID); uploadErr != nil {
			return nil, uploadErr
		}
		resolved, resolveErr := s.catalog.ResolveSource(ctx, actor, opts.Source)
		if resolveErr != nil {
			return nil, mapAdapterError(resolveErr)
		}
		opts.Source = resolved
	}
	permission, err := changePermission(operation)
	if err != nil {
		return nil, err
	}
	if !s.auth.Allowed(ctx, actor, permission, cluster, opts.Namespace) {
		return nil, bizerr.ErrForbidden
	}
	if err := validateChangeOptions(&opts); err != nil {
		return nil, err
	}
	preview, err := s.adapter.Preview(ctx, operation, cluster, opts)
	if err != nil {
		return nil, mapAdapterError(err)
	}
	if opts.CreateNamespace {
		preview.Risks = append(preview.Risks, Risk{Level: "critical", Code: "namespace-create", Message: "creates the target namespace outside the rendered manifest", Resource: "Namespace/" + opts.Namespace})
	}
	preview.Operation = operation
	preview.CanExecute = !hasCriticalRisk(preview.Risks) || actor.Role == "admin" || actor.Role == "super-admin"
	if preview.CanExecute && !actor.PreviewOnly {
		token, tokenErr := randomToken()
		if tokenErr != nil {
			return nil, bizerr.New(bizerr.CodeInternal, "could not create confirmation")
		}
		expires := time.Now().Add(10 * time.Minute)
		s.mu.Lock()
		s.prunePreviewsLocked(time.Now())
		if len(s.previews) >= 5000 {
			s.mu.Unlock()
			return nil, bizerr.New(bizerr.CodeConflict, "too many pending package previews; try again later")
		}
		s.previews[token] = previewGrant{ActorID: actor.UserID, Operation: operation, Cluster: cluster, Fingerprint: changeFingerprint(opts), Digest: preview.Digest, ExpiresAt: expires}
		s.mu.Unlock()
		preview.ConfirmationToken, preview.ExpiresAt = token, expires
	}
	preview.Manifest = redactManifest(preview.Manifest)
	return preview, nil
}

func (s *Service) Install(ctx context.Context, actor Actor, cluster string, opts ChangeOptions) error {
	return s.executeChange(ctx, actor, "install", cluster, opts)
}

func (s *Service) Upgrade(ctx context.Context, actor Actor, cluster string, opts ChangeOptions) error {
	return s.executeChange(ctx, actor, "upgrade", cluster, opts)
}

func (s *Service) executeChange(ctx context.Context, actor Actor, operation, cluster string, opts ChangeOptions) error {
	trackedSource := opts.Source
	managedRepository := opts.Source.RepositoryID != 0
	if managedRepository && !isAdmin(actor) {
		return bizerr.ErrForbidden
	}
	if s.catalog != nil {
		if uploadErr := s.catalog.AuthorizeUpload(actor, opts.Source.UploadID); uploadErr != nil {
			return uploadErr
		}
		resolved, resolveErr := s.catalog.ResolveSource(ctx, actor, opts.Source)
		if resolveErr != nil {
			return mapAdapterError(resolveErr)
		}
		opts.Source = resolved
	}
	permission, _ := changePermission(operation)
	if !s.auth.Allowed(ctx, actor, permission, cluster, opts.Namespace) {
		return bizerr.ErrForbidden
	}
	if err := validateChangeOptions(&opts); err != nil {
		return err
	}
	s.mu.Lock()
	grant, ok := s.previews[opts.ConfirmationToken]
	delete(s.previews, opts.ConfirmationToken)
	s.mu.Unlock()
	if !ok || time.Now().After(grant.ExpiresAt) || grant.ActorID != actor.UserID || grant.Operation != operation || grant.Cluster != cluster || grant.Fingerprint != changeFingerprint(opts) {
		return bizerr.New(bizerr.CodeValidation, "preview expired or does not match this operation; preview again")
	}
	opts.ExpectedDigest = grant.Digest
	summary := fmt.Sprintf("chart=%s,version=%s,wait=%t,atomic=%t,timeout=%s", opts.Source.Chart, opts.Source.Version, opts.Wait, opts.Atomic, opts.Timeout)
	err := s.mutate(ctx, actor, operation, cluster, opts.Namespace, opts.ReleaseName, 0, summary, func() error {
		if operation == "install" {
			return s.adapter.Install(ctx, cluster, opts)
		}
		return s.adapter.Upgrade(ctx, cluster, opts)
	})
	if err == nil && s.catalog != nil {
		_ = s.catalog.SaveReleaseSource(ctx, cluster, opts.Namespace, opts.ReleaseName, trackedSource)
	}
	return err
}

func (s *Service) CheckUpgrade(ctx context.Context, actor Actor, cluster, namespace, name string, provided *ChartSource) (*UpgradeCandidate, error) {
	if !s.auth.Allowed(ctx, actor, PermissionUpgrade, cluster, namespace) {
		return nil, bizerr.ErrForbidden
	}
	if s.catalog == nil {
		return nil, bizerr.New(bizerr.CodeInternal, "Helm catalog is unavailable")
	}
	release, err := s.adapter.Get(ctx, cluster, namespace, name, 0)
	if err != nil {
		return nil, mapAdapterError(err)
	}
	source := ChartSource{}
	found := false
	if provided != nil {
		source = *provided
		found = true
	} else {
		source, found, err = s.catalog.ReleaseSource(ctx, cluster, namespace, name)
		if err != nil {
			return nil, mapAdapterError(err)
		}
	}
	result := &UpgradeCandidate{SourceRequired: !found, CurrentVersion: release.ChartVersion}
	if !found {
		return result, nil
	}
	source.Chart = strings.TrimSpace(source.Chart)
	if source.Chart == "" || source.Chart != release.Chart {
		return nil, bizerr.New(bizerr.CodeValidation, "chart source must match the installed release chart")
	}
	latest, err := s.catalog.LatestVersion(ctx, actor, source)
	if err != nil {
		return nil, mapAdapterError(err)
	}
	currentVersion, currentErr := semver.NewVersion(release.ChartVersion)
	latestVersion, latestErr := semver.NewVersion(latest.Version)
	if currentErr != nil || latestErr != nil {
		return nil, bizerr.New(bizerr.CodeValidation, "installed and repository chart versions must use semantic versioning")
	}
	if provided != nil {
		if err := s.catalog.SaveReleaseSource(ctx, cluster, namespace, name, source); err != nil {
			return nil, mapAdapterError(err)
		}
	}
	source.Version = latest.Version
	result.SourceRequired = false
	result.Available = latestVersion.GreaterThan(currentVersion)
	result.LatestVersion = latest.Version
	result.AppVersion = latest.AppVersion
	result.Source = source
	return result, nil
}

func changePermission(operation string) (string, error) {
	switch operation {
	case "install":
		return PermissionInstall, nil
	case "upgrade":
		return PermissionUpgrade, nil
	}
	return "", bizerr.New(bizerr.CodeParamInvalid, "operation must be install or upgrade")
}

func validateChangeOptions(opts *ChangeOptions) error {
	opts.ReleaseName, opts.Namespace, opts.Source.Chart = strings.TrimSpace(opts.ReleaseName), strings.TrimSpace(opts.Namespace), strings.TrimSpace(opts.Source.Chart)
	if opts.ReleaseName == "" || opts.Namespace == "" || opts.Source.Chart == "" && opts.Source.UploadID == "" {
		return bizerr.New(bizerr.CodeParamInvalid, "release name, namespace, and chart are required")
	}
	if len(opts.ReleaseName) > 53 || len(opts.Namespace) > 63 || len(opts.Source.Chart) > 2048 || len(opts.Source.RepoURL) > 2048 {
		return bizerr.New(bizerr.CodeParamInvalid, "package input is too long")
	}
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Minute
	}
	if opts.Timeout < time.Second || opts.Timeout > 30*time.Minute {
		return bizerr.New(bizerr.CodeParamInvalid, "timeout must be between 1 second and 30 minutes")
	}
	encoded, err := json.Marshal(opts.Values)
	if err != nil || len(encoded) > 512*1024 {
		return bizerr.New(bizerr.CodeParamInvalid, "values must be valid JSON and no larger than 512 KiB")
	}
	return nil
}

func changeFingerprint(opts ChangeOptions) string {
	copy := opts
	copy.ConfirmationToken = ""
	copy.ExpectedDigest = ""
	b, _ := json.Marshal(copy)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func hasCriticalRisk(risks []Risk) bool {
	for _, risk := range risks {
		if risk.Level == "critical" {
			return true
		}
	}
	return false
}
func (s *Service) prunePreviewsLocked(now time.Time) {
	for token, grant := range s.previews {
		if now.After(grant.ExpiresAt) {
			delete(s.previews, token)
		}
	}
}

func (s *Service) List(ctx context.Context, actor Actor, cluster string, opts ListOptions) ([]Release, error) {
	if !s.auth.Allowed(ctx, actor, PermissionRead, cluster, opts.Namespace) {
		return nil, bizerr.ErrForbidden
	}
	if opts.Limit == 0 {
		opts.Limit = 100
	}
	if opts.Limit < 1 || opts.Limit > 200 {
		return nil, bizerr.New(bizerr.CodeParamInvalid, "limit must be between 1 and 200")
	}
	items, err := s.adapter.List(ctx, cluster, opts)
	if err != nil {
		return nil, mapAdapterError(err)
	}
	for i := range items {
		sanitizeRelease(&items[i])
	}
	return items, nil
}

func (s *Service) Get(ctx context.Context, actor Actor, cluster, namespace, name string, revision int) (*Release, error) {
	if !s.auth.Allowed(ctx, actor, PermissionRead, cluster, namespace) {
		return nil, bizerr.ErrForbidden
	}
	r, err := s.adapter.Get(ctx, cluster, namespace, name, revision)
	if err != nil {
		return nil, mapAdapterError(err)
	}
	if s.catalog != nil && revision == 0 {
		if source, found, sourceErr := s.catalog.ReleaseSource(ctx, cluster, namespace, name); sourceErr == nil && found {
			r.Source = &source
		}
	}
	sanitizeRelease(r)
	return r, nil
}

func (s *Service) History(ctx context.Context, actor Actor, cluster, namespace, name string) ([]Release, error) {
	if !s.auth.Allowed(ctx, actor, PermissionRead, cluster, namespace) {
		return nil, bizerr.ErrForbidden
	}
	items, err := s.adapter.History(ctx, cluster, namespace, name)
	if err != nil {
		return nil, mapAdapterError(err)
	}
	for i := range items {
		sanitizeRelease(&items[i])
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Revision > items[j].Revision })
	return items, nil
}

func (s *Service) Rollback(ctx context.Context, actor Actor, cluster, namespace, name string, opts RollbackOptions) (err error) {
	if !s.auth.Allowed(ctx, actor, PermissionRollback, cluster, namespace) {
		return bizerr.ErrForbidden
	}
	if opts.Revision < 1 {
		return bizerr.New(bizerr.CodeParamInvalid, "revision must be positive")
	}
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Minute
	}
	if opts.Timeout < time.Second || opts.Timeout > 30*time.Minute {
		return bizerr.New(bizerr.CodeParamInvalid, "timeout must be between 1 second and 30 minutes")
	}
	release, getErr := s.adapter.Get(ctx, cluster, namespace, name, opts.Revision)
	if getErr != nil {
		return mapAdapterError(getErr)
	}
	if release.Revision != opts.Revision {
		return bizerr.ErrNotFound
	}
	return s.mutate(ctx, actor, "rollback", cluster, namespace, name, opts.Revision,
		fmt.Sprintf("wait=%t,atomic=%t,timeout=%s", opts.Wait, opts.Atomic, opts.Timeout),
		func() error { return s.adapter.Rollback(ctx, cluster, namespace, name, opts) })
}

func (s *Service) Remove(ctx context.Context, actor Actor, cluster, namespace, name string, opts RemoveOptions) error {
	if !s.auth.Allowed(ctx, actor, PermissionRemove, cluster, namespace) {
		return bizerr.ErrForbidden
	}
	if opts.Confirmation != name {
		return bizerr.New(bizerr.CodeValidation, "confirmation must exactly match the release name")
	}
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Minute
	}
	if opts.Timeout < time.Second || opts.Timeout > 30*time.Minute {
		return bizerr.New(bizerr.CodeParamInvalid, "timeout must be between 1 second and 30 minutes")
	}
	err := s.mutate(ctx, actor, "remove", cluster, namespace, name, 0,
		fmt.Sprintf("keepHistory=%t,wait=%t,timeout=%s", opts.KeepHistory, opts.Wait, opts.Timeout),
		func() error { return s.adapter.Remove(ctx, cluster, namespace, name, opts) })
	if err == nil && s.catalog != nil {
		_ = s.catalog.DeleteReleaseSource(ctx, cluster, namespace, name)
	}
	return err
}

func (s *Service) mutate(ctx context.Context, actor Actor, action, cluster, namespace, name string, revision int, summary string, fn func() error) (err error) {
	key := cluster + "\x00" + namespace + "\x00" + name
	s.mu.Lock()
	if _, exists := s.active[key]; exists {
		s.mu.Unlock()
		return bizerr.New(bizerr.CodeConflict, "another release operation is already running")
	}
	s.active[key] = struct{}{}
	s.mu.Unlock()
	start := time.Now()
	defer func() {
		s.mu.Lock()
		delete(s.active, key)
		s.mu.Unlock()
		outcome := "succeeded"
		if err != nil {
			outcome = "failed"
		}
		if s.audit != nil {
			s.audit.RecordPackageAudit(AuditEvent{Actor: actor, Action: action, Cluster: cluster, Namespace: namespace, Release: name, Revision: revision, Options: summary, Outcome: outcome, Duration: time.Since(start)})
		}
	}()
	if err = fn(); err != nil {
		err = mapAdapterError(err)
	}
	return err
}

func mapAdapterError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*bizerr.BizError); ok {
		return err
	}
	if message, ok := chartValidationMessage(err); ok {
		return bizerr.New(bizerr.CodeValidation, "chart values are invalid: "+message)
	}
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "cannot re-use a name that is still in use") {
		return bizerr.New(bizerr.CodeConflict, "release already exists; upgrade or remove it before reinstalling")
	}
	if strings.Contains(text, "not found") {
		return bizerr.New(bizerr.CodeNotFound, "release or revision not found")
	}
	detail := safeAdapterErrorDetail(err)
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(text, "timed out") || strings.Contains(text, "timeout") {
		return bizerr.New(bizerr.CodeK8sUnavailable, "timed out waiting for release resources; inspect Pods and Events, then retry with a longer timeout")
	}
	if strings.Contains(text, "failed pre-install") || strings.Contains(text, "failed post-install") || strings.Contains(text, "failed pre-upgrade") || strings.Contains(text, "failed post-upgrade") {
		return bizerr.New(bizerr.CodeValidation, "Helm hook failed: "+detail)
	}
	if strings.Contains(text, "unable to build kubernetes objects") || strings.Contains(text, "rendered manifests contain") || strings.Contains(text, "cannot patch") || strings.Contains(text, "is invalid") {
		return bizerr.New(bizerr.CodeValidation, "Kubernetes rejected the rendered resources: "+detail)
	}
	if strings.Contains(text, "connection refused") || strings.Contains(text, "no route to host") || strings.Contains(text, "tls handshake") || strings.Contains(text, "server is currently unable") {
		return bizerr.New(bizerr.CodeK8sUnavailable, "cannot reach the Kubernetes API: "+detail)
	}
	return bizerr.New(bizerr.CodeK8sUnavailable, "package operation failed: "+detail)
}

var (
	adapterCredentialPattern = regexp.MustCompile(`(?i)(password|token|secret|authorization|api[-_]?key|client[-_]?key[-_]?data)(\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`)
	adapterPrivateKeyPattern = regexp.MustCompile(`(?is)-----BEGIN[^-]*PRIVATE KEY-----.*?-----END[^-]*PRIVATE KEY-----`)
	adapterURLUserPattern    = regexp.MustCompile(`://[^/@\s]+@`)
)

func safeAdapterErrorDetail(err error) string {
	detail := adapterPrivateKeyPattern.ReplaceAllString(err.Error(), "[REDACTED PRIVATE KEY]")
	detail = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(detail, "\r", " "), "\n", " "))
	detail = adapterCredentialPattern.ReplaceAllString(detail, "$1$2[REDACTED]")
	detail = adapterURLUserPattern.ReplaceAllString(detail, "://[REDACTED]@")
	runes := []rune(detail)
	if len(runes) > 512 {
		detail = string(runes[:512])
	}
	if detail == "" {
		return "unknown Helm error"
	}
	return detail
}

func chartValidationMessage(err error) (string, bool) {
	const (
		prefix = "execution error at ("
		marker = "): "
	)
	text := err.Error()
	start := strings.LastIndex(text, prefix)
	if start < 0 {
		return "", false
	}
	detailStart := strings.Index(text[start:], marker)
	if detailStart < 0 {
		return "", false
	}
	detail := strings.TrimSpace(text[start+detailStart+len(marker):])
	if newline := strings.IndexAny(detail, "\r\n"); newline >= 0 {
		detail = strings.TrimSpace(detail[:newline])
	}
	if detail == "" {
		return "", false
	}
	if len(detail) > 512 {
		detail = detail[:512]
	}
	return detail, true
}

func sanitizeRelease(r *Release) {
	r.Values = redactMap(r.Values)
	for i := range r.Resources {
		if r.Resources[i].Kind == "Secret" {
			r.Resources[i].Name = "[REDACTED]"
		}
	}
}

func redactMap(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return nil
	}
	out := make(map[string]interface{}, len(input))
	for k, v := range input {
		lower := strings.ToLower(k)
		if strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "token") || strings.Contains(lower, "privatekey") || strings.Contains(lower, "certificate") {
			out[k] = "[REDACTED]"
			continue
		}
		switch value := v.(type) {
		case map[string]interface{}:
			out[k] = redactMap(value)
		case []interface{}:
			copyValues := make([]interface{}, len(value))
			for i, item := range value {
				if nested, ok := item.(map[string]interface{}); ok {
					copyValues[i] = redactMap(nested)
				} else {
					copyValues[i] = item
				}
			}
			out[k] = copyValues
		default:
			out[k] = value
		}
	}
	return out
}
