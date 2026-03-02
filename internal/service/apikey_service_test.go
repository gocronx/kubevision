package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kubevision/kubevision/internal/model"
)

// ---------------------------------------------------------------------------
// Mock APIKeyRepo
// ---------------------------------------------------------------------------

type mockAPIKeyRepo struct {
	keys    map[uint]*model.APIKey
	byHash  map[string]*model.APIKey
	nextID  uint
	createErr error
	deleteErr error
}

func newMockAPIKeyRepo() *mockAPIKeyRepo {
	return &mockAPIKeyRepo{
		keys:   make(map[uint]*model.APIKey),
		byHash: make(map[string]*model.APIKey),
		nextID: 1,
	}
}

func (m *mockAPIKeyRepo) Create(_ context.Context, key *model.APIKey) error {
	if m.createErr != nil {
		return m.createErr
	}
	key.ID = m.nextID
	key.CreatedAt = time.Now()
	m.nextID++
	cp := *key
	m.keys[key.ID] = &cp
	m.byHash[key.KeyHash] = &cp
	return nil
}

func (m *mockAPIKeyRepo) GetByKeyHash(_ context.Context, hash string) (*model.APIKey, error) {
	k, ok := m.byHash[hash]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *k
	return &cp, nil
}

func (m *mockAPIKeyRepo) ListByUser(_ context.Context, userID uint) ([]model.APIKey, error) {
	var result []model.APIKey
	for _, k := range m.keys {
		if k.UserID == userID {
			result = append(result, *k)
		}
	}
	return result, nil
}

func (m *mockAPIKeyRepo) Delete(_ context.Context, id uint) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if k, ok := m.keys[id]; ok {
		delete(m.byHash, k.KeyHash)
	}
	delete(m.keys, id)
	return nil
}

// ---------------------------------------------------------------------------
// Mock UserRepo (reuse approach)
// ---------------------------------------------------------------------------

type mockUserRepoForAPIKey struct {
	users map[uint]*model.User
}

func newMockUserRepoForAPIKey() *mockUserRepoForAPIKey {
	return &mockUserRepoForAPIKey{users: make(map[uint]*model.User)}
}

func (m *mockUserRepoForAPIKey) addUser(u *model.User) { m.users[u.ID] = u }

func (m *mockUserRepoForAPIKey) Create(_ context.Context, u *model.User) error { return nil }
func (m *mockUserRepoForAPIKey) GetByID(_ context.Context, id uint) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}
func (m *mockUserRepoForAPIKey) GetByUsername(_ context.Context, _ string) (*model.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUserRepoForAPIKey) Update(_ context.Context, _ *model.User) error { return nil }
func (m *mockUserRepoForAPIKey) Delete(_ context.Context, _ uint) error         { return nil }
func (m *mockUserRepoForAPIKey) List(_ context.Context) ([]model.User, error)   { return nil, nil }
func (m *mockUserRepoForAPIKey) GetByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUserRepoForAPIKey) GetByOAuthID(_ context.Context, _, _ string) (*model.User, error) {
	return nil, errors.New("not implemented")
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestAPIKeyService_Generate(t *testing.T) {
	keyRepo := newMockAPIKeyRepo()
	userRepo := newMockUserRepoForAPIKey()
	userRepo.addUser(&model.User{ID: 1, Username: "admin"})

	svc := NewAPIKeyService(keyRepo, userRepo)
	ctx := context.Background()

	resp, err := svc.Generate(ctx, 1, "my-key", nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.HasPrefix(resp.PlainKey, "kv_") {
		t.Errorf("key should start with 'kv_', got %q", resp.PlainKey[:10])
	}
	if resp.Name != "my-key" {
		t.Errorf("expected name 'my-key', got %q", resp.Name)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestAPIKeyService_Generate_UserNotFound(t *testing.T) {
	keyRepo := newMockAPIKeyRepo()
	userRepo := newMockUserRepoForAPIKey()

	svc := NewAPIKeyService(keyRepo, userRepo)
	ctx := context.Background()

	_, err := svc.Generate(ctx, 999, "my-key", nil)
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestAPIKeyService_Validate(t *testing.T) {
	keyRepo := newMockAPIKeyRepo()
	userRepo := newMockUserRepoForAPIKey()
	userRepo.addUser(&model.User{ID: 1, Username: "admin"})

	svc := NewAPIKeyService(keyRepo, userRepo)
	ctx := context.Background()

	resp, err := svc.Generate(ctx, 1, "validate-key", nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	record, err := svc.Validate(ctx, resp.PlainKey)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if record.UserID != 1 {
		t.Errorf("expected UserID 1, got %d", record.UserID)
	}
}

func TestAPIKeyService_Validate_InvalidKey(t *testing.T) {
	keyRepo := newMockAPIKeyRepo()
	userRepo := newMockUserRepoForAPIKey()

	svc := NewAPIKeyService(keyRepo, userRepo)
	ctx := context.Background()

	_, err := svc.Validate(ctx, "kv_invalid_key_here")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestAPIKeyService_Validate_ExpiredKey(t *testing.T) {
	keyRepo := newMockAPIKeyRepo()
	userRepo := newMockUserRepoForAPIKey()
	userRepo.addUser(&model.User{ID: 1, Username: "admin"})

	svc := NewAPIKeyService(keyRepo, userRepo)
	ctx := context.Background()

	expired := time.Now().Add(-1 * time.Hour)
	resp, err := svc.Generate(ctx, 1, "expired-key", &expired)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	_, err = svc.Validate(ctx, resp.PlainKey)
	if err == nil {
		t.Fatal("expected error for expired key")
	}
}

func TestAPIKeyService_ListByUser(t *testing.T) {
	keyRepo := newMockAPIKeyRepo()
	userRepo := newMockUserRepoForAPIKey()
	userRepo.addUser(&model.User{ID: 1, Username: "admin"})

	svc := NewAPIKeyService(keyRepo, userRepo)
	ctx := context.Background()

	// Generate 2 keys.
	_, _ = svc.Generate(ctx, 1, "key-1", nil)
	_, _ = svc.Generate(ctx, 1, "key-2", nil)

	infos, err := svc.ListByUser(ctx, 1)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(infos) != 2 {
		t.Errorf("expected 2 keys, got %d", len(infos))
	}
}

func TestAPIKeyService_Revoke(t *testing.T) {
	keyRepo := newMockAPIKeyRepo()
	userRepo := newMockUserRepoForAPIKey()
	userRepo.addUser(&model.User{ID: 1, Username: "admin"})

	svc := NewAPIKeyService(keyRepo, userRepo)
	ctx := context.Background()

	resp, _ := svc.Generate(ctx, 1, "revoke-key", nil)

	err := svc.Revoke(ctx, resp.ID, 1)
	if err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	// Validate should fail now.
	_, err = svc.Validate(ctx, resp.PlainKey)
	if err == nil {
		t.Fatal("expected error after revoke")
	}
}

func TestAPIKeyService_Revoke_NotOwned(t *testing.T) {
	keyRepo := newMockAPIKeyRepo()
	userRepo := newMockUserRepoForAPIKey()
	userRepo.addUser(&model.User{ID: 1, Username: "admin"})
	userRepo.addUser(&model.User{ID: 2, Username: "other"})

	svc := NewAPIKeyService(keyRepo, userRepo)
	ctx := context.Background()

	resp, _ := svc.Generate(ctx, 1, "owned-by-1", nil)

	// User 2 tries to revoke user 1's key.
	err := svc.Revoke(ctx, resp.ID, 2)
	if err == nil {
		t.Fatal("expected error when revoking key not owned by user")
	}
}

// Verify SHA256 hash consistency.
func TestAPIKeyService_HashConsistency(t *testing.T) {
	plainKey := "kv_abcdef1234567890"
	sum := sha256.Sum256([]byte(plainKey))
	hash := hex.EncodeToString(sum[:])

	if len(hash) != 64 {
		t.Errorf("SHA256 hex should be 64 chars, got %d", len(hash))
	}
}
