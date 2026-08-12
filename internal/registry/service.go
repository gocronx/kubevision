package registry

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type tagClient interface {
	Tags(context.Context, Reference, int, string) ([]string, string, error)
}

type Page struct {
	Reference  Reference `json:"reference"`
	Tags       []string  `json:"tags"`
	NextCursor string    `json:"nextCursor,omitempty"`
	Cached     bool      `json:"cached"`
}
type cacheEntry struct {
	page    Page
	expires time.Time
	touched time.Time
}

type Service struct {
	client     tagClient
	ttl        time.Duration
	maxEntries int
	maxTags    int
	now        func() time.Time
	mu         sync.Mutex
	cache      map[string]cacheEntry
}

func NewService(client tagClient, ttl time.Duration, maxEntries, maxTags int) *Service {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	if maxEntries <= 0 {
		maxEntries = 128
	}
	if maxTags <= 0 {
		maxTags = 100
	}
	return &Service{client: client, ttl: ttl, maxEntries: maxEntries, maxTags: maxTags, now: time.Now, cache: make(map[string]cacheEntry)}
}

func (s *Service) Discover(ctx context.Context, image, prefix string, limit int, cursor string) (Page, error) {
	ref, err := ParseReference(image)
	if err != nil {
		return Page{}, err
	}
	if limit <= 0 {
		limit = 25
	}
	if limit > s.maxTags {
		limit = s.maxTags
	}
	if len(prefix) > 128 || len(cursor) > 128 {
		return Page{}, fmt.Errorf("filter or cursor is too long")
	}
	key := strings.Join([]string{ref.Registry, ref.Repository, "anonymous", prefix, cursor, fmt.Sprint(limit)}, "\x00")
	if page, ok := s.load(key); ok {
		page.Cached = true
		return page, nil
	}
	tags, next, err := s.client.Tags(ctx, ref, limit, cursor)
	if err != nil {
		return Page{}, err
	}
	filtered := tags[:0]
	for _, tag := range tags {
		if prefix == "" || strings.HasPrefix(tag, prefix) {
			filtered = append(filtered, tag)
		}
	}
	sort.Strings(filtered)
	page := Page{Reference: ref, Tags: filtered, NextCursor: next}
	s.store(key, page)
	return page, nil
}

func (s *Service) load(key string) (Page, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cache[key]
	if !ok {
		return Page{}, false
	}
	now := s.now()
	if !entry.expires.After(now) {
		delete(s.cache, key)
		return Page{}, false
	}
	entry.touched = now
	s.cache[key] = entry
	return entry.page, true
}
func (s *Service) store(key string, page Page) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for k, v := range s.cache {
		if !v.expires.After(now) {
			delete(s.cache, k)
		}
	}
	if len(s.cache) >= s.maxEntries {
		oldestKey := ""
		var oldest time.Time
		for k, v := range s.cache {
			if oldestKey == "" || v.touched.Before(oldest) {
				oldestKey, oldest = k, v.touched
			}
		}
		delete(s.cache, oldestKey)
	}
	s.cache[key] = cacheEntry{page: page, expires: now.Add(s.ttl), touched: now}
}
