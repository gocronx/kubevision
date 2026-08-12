package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gocronx/kubevision/internal/model"
)

// ---------------------------------------------------------------------------
// Mock WebhookRepo
// ---------------------------------------------------------------------------

type mockWebhookRepo struct {
	webhooks map[uint]*model.Webhook
	nextID   uint
}

func newMockWebhookRepo() *mockWebhookRepo {
	return &mockWebhookRepo{
		webhooks: make(map[uint]*model.Webhook),
		nextID:   1,
	}
}

func (m *mockWebhookRepo) Create(_ context.Context, wh *model.Webhook) error {
	wh.ID = m.nextID
	m.nextID++
	cp := *wh
	m.webhooks[wh.ID] = &cp
	return nil
}

func (m *mockWebhookRepo) GetByID(_ context.Context, id uint) (*model.Webhook, error) {
	wh, ok := m.webhooks[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *wh
	return &cp, nil
}

func (m *mockWebhookRepo) Update(_ context.Context, wh *model.Webhook) error {
	if _, ok := m.webhooks[wh.ID]; !ok {
		return errors.New("not found")
	}
	cp := *wh
	m.webhooks[wh.ID] = &cp
	return nil
}

func (m *mockWebhookRepo) Delete(_ context.Context, id uint) error {
	delete(m.webhooks, id)
	return nil
}

func (m *mockWebhookRepo) List(_ context.Context) ([]model.Webhook, error) {
	var result []model.Webhook
	for _, wh := range m.webhooks {
		result = append(result, *wh)
	}
	return result, nil
}

func (m *mockWebhookRepo) ListActive(_ context.Context) ([]model.Webhook, error) {
	var result []model.Webhook
	for _, wh := range m.webhooks {
		if wh.IsActive {
			result = append(result, *wh)
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestWebhookService_CreateAndList(t *testing.T) {
	repo := newMockWebhookRepo()
	svc := NewWebhookService(repo, nil)
	ctx := context.Background()

	resp, err := svc.Create(ctx, &WebhookRequest{
		Name:     "test-hook",
		URL:      "https://8.8.8.8/webhook",
		Events:   []string{"create", "delete"},
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.Name != "test-hook" {
		t.Errorf("expected name 'test-hook', got %q", resp.Name)
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(list))
	}
}

func TestWebhookService_Update(t *testing.T) {
	repo := newMockWebhookRepo()
	svc := NewWebhookService(repo, nil)
	ctx := context.Background()

	resp, err := svc.Create(ctx, &WebhookRequest{Name: "hook1", URL: "https://8.8.8.8/old", IsActive: true})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := svc.Update(ctx, resp.ID, &WebhookRequest{
		Name:     "hook1",
		URL:      "https://1.1.1.1/new",
		IsActive: false,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.URL != "https://1.1.1.1/new" {
		t.Errorf("expected updated URL, got %q", updated.URL)
	}
	if updated.IsActive {
		t.Error("expected IsActive to be false")
	}
}

func TestWebhookService_Delete(t *testing.T) {
	repo := newMockWebhookRepo()
	svc := NewWebhookService(repo, nil)
	ctx := context.Background()

	resp, createErr := svc.Create(ctx, &WebhookRequest{Name: "delete-me", URL: "https://8.8.4.4/delete"})
	if createErr != nil {
		t.Fatalf("Create failed: %v", createErr)
	}

	err := svc.Delete(ctx, resp.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = svc.GetByID(ctx, resp.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestWebhookService_Delete_NotFound(t *testing.T) {
	repo := newMockWebhookRepo()
	svc := NewWebhookService(repo, nil)
	ctx := context.Background()

	err := svc.Delete(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent webhook")
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		slice  []string
		target string
		want   bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{nil, "a", false},
	}

	for _, tc := range tests {
		got := containsString(tc.slice, tc.target)
		if got != tc.want {
			t.Errorf("containsString(%v, %q) = %v, want %v", tc.slice, tc.target, got, tc.want)
		}
	}
}

func TestComputeHMAC(t *testing.T) {
	sig := computeHMAC([]byte("hello"), "secret")
	if sig == "" {
		t.Error("expected non-empty HMAC signature")
	}
	if len(sig) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(sig))
	}

	// Same input should produce same output.
	sig2 := computeHMAC([]byte("hello"), "secret")
	if sig != sig2 {
		t.Error("HMAC should be deterministic")
	}

	// Different secret should produce different output.
	sig3 := computeHMAC([]byte("hello"), "other-secret")
	if sig == sig3 {
		t.Error("different secrets should produce different signatures")
	}
}
