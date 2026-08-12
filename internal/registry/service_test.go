package registry

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeTagClient struct {
	calls int
	tags  []string
	next  string
	err   error
}

func (f *fakeTagClient) Tags(context.Context, Reference, int, string) ([]string, string, error) {
	f.calls++
	return append([]string(nil), f.tags...), f.next, f.err
}

func TestServicePaginationPrefixAndCache(t *testing.T) {
	client := &fakeTagClient{tags: []string{"v2", "dev", "v1"}, next: "v2"}
	svc := NewService(client, time.Minute, 2, 10)
	page, err := svc.Discover(context.Background(), "nginx", "v", 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Tags) != 2 || page.Tags[0] != "v1" || page.NextCursor != "v2" || page.Cached {
		t.Fatalf("unexpected page: %#v", page)
	}
	page, err = svc.Discover(context.Background(), "nginx", "v", 3, "")
	if err != nil || !page.Cached || client.calls != 1 {
		t.Fatalf("cache miss: page=%#v calls=%d err=%v", page, client.calls, err)
	}
}

func TestServiceDoesNotCacheAuthenticationFailure(t *testing.T) {
	client := &fakeTagClient{err: ErrAuthentication}
	svc := NewService(client, time.Minute, 2, 10)
	for range 2 {
		if _, err := svc.Discover(context.Background(), "nginx", "", 10, ""); !errors.Is(err, ErrAuthentication) {
			t.Fatalf("got %v", err)
		}
	}
	if client.calls != 2 {
		t.Fatalf("authentication failure was cached")
	}
}

func TestServiceBoundsEntriesAndLimit(t *testing.T) {
	client := &fakeTagClient{tags: []string{"one"}}
	svc := NewService(client, time.Minute, 1, 5)
	if _, err := svc.Discover(context.Background(), "nginx", "", 999, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Discover(context.Background(), "busybox", "", 999, ""); err != nil {
		t.Fatal(err)
	}
	if len(svc.cache) != 1 {
		t.Fatalf("cache has %d entries", len(svc.cache))
	}
}
