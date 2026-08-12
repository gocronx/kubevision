package packages

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
)

type Service struct {
	adapter Adapter
	auth    Authorizer
	audit   Auditor
	mu      sync.Mutex
	active  map[string]struct{}
}

func NewService(adapter Adapter, auth Authorizer, audit Auditor) *Service {
	return &Service{adapter: adapter, auth: auth, audit: audit, active: make(map[string]struct{})}
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
	return s.mutate(ctx, actor, "remove", cluster, namespace, name, 0,
		fmt.Sprintf("keepHistory=%t,wait=%t,timeout=%s", opts.KeepHistory, opts.Wait, opts.Timeout),
		func() error { return s.adapter.Remove(ctx, cluster, namespace, name, opts) })
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
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "not found") {
		return bizerr.New(bizerr.CodeNotFound, "release or revision not found")
	}
	return bizerr.New(bizerr.CodeK8sUnavailable, "package operation failed")
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
