package cli

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
)

func newTestUserRepo(t *testing.T) repository.UserRepo {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return repository.NewUserRepo(db)
}

func TestResetUserPassword(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	hash, _ := auth.HashPassword("oldpass1")
	if err := repo.Create(ctx, &model.User{Username: "alice", PasswordHash: hash, Role: "viewer", TokenVersion: 3}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := resetUserPassword(ctx, repo, "alice", "newpass1"); err != nil {
		t.Fatalf("reset: %v", err)
	}

	u, _ := repo.GetByUsername(ctx, "alice")
	if !auth.CheckPassword("newpass1", u.PasswordHash) {
		t.Fatal("new password does not verify")
	}
	if auth.CheckPassword("oldpass1", u.PasswordHash) {
		t.Fatal("old password should no longer verify")
	}
	if u.TokenVersion != 4 {
		t.Fatalf("token version = %d, want 4 (bumped to invalidate sessions)", u.TokenVersion)
	}
}

func TestResetUserPassword_Errors(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	if err := resetUserPassword(ctx, repo, "ghost", "longenough"); err == nil {
		t.Fatal("expected error for missing user")
	}
	hash, _ := auth.HashPassword("x")
	_ = repo.Create(ctx, &model.User{Username: "bob", PasswordHash: hash, Role: "viewer"})
	if err := resetUserPassword(ctx, repo, "bob", "short"); err == nil {
		t.Fatal("expected error for too-short password")
	}
}

func TestCreateUser(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	if err := createUser(ctx, repo, "carol", "secret1", "editor", "carol@example.com"); err != nil {
		t.Fatalf("create: %v", err)
	}
	u, _ := repo.GetByUsername(ctx, "carol")
	if u == nil || u.Role != "editor" || !u.IsActive || u.AuthProvider != "local" {
		t.Fatalf("unexpected user: %+v", u)
	}
	if !auth.CheckPassword("secret1", u.PasswordHash) {
		t.Fatal("password does not verify")
	}

	// Duplicate username is rejected.
	if err := createUser(ctx, repo, "carol", "secret2", "viewer", ""); err == nil {
		t.Fatal("expected duplicate-username error")
	}
	// Too-short password is rejected.
	if err := createUser(ctx, repo, "dave", "no", "viewer", ""); err == nil {
		t.Fatal("expected too-short password error")
	}
}
