package cli

import (
	"context"
	"testing"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/model"
	"github.com/gocronx/kubevision/internal/repository"
)

func seedUser(t *testing.T, repo repository.UserRepo, u *model.User) {
	t.Helper()
	if u.PasswordHash == "" {
		u.PasswordHash, _ = auth.HashPassword("password1")
	}
	if err := repo.Create(context.Background(), u); err != nil {
		t.Fatalf("seed %s: %v", u.Username, err)
	}
}

func TestClear2FA(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()
	seedUser(t, repo, &model.User{Username: "a", Role: "admin", TOTPEnabled: true, TOTPSecretEnc: "secret", RecoveryCodesEnc: "codes"})

	if err := clear2FA(ctx, repo, "a"); err != nil {
		t.Fatalf("clear2FA: %v", err)
	}
	u, _ := repo.GetByUsername(ctx, "a")
	if u.TOTPEnabled || u.TOTPSecretEnc != "" || u.RecoveryCodesEnc != "" {
		t.Fatalf("2FA not cleared: %+v", u)
	}
	if err := clear2FA(ctx, repo, "ghost"); err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestSetUserRole(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()
	seedUser(t, repo, &model.User{Username: "u", Role: "viewer", IsActive: true, TokenVersion: 1})

	if err := setUserRole(ctx, repo, "u", "admin"); err != nil {
		t.Fatalf("setUserRole: %v", err)
	}
	u, _ := repo.GetByUsername(ctx, "u")
	if u.Role != "admin" || u.TokenVersion != 2 {
		t.Fatalf("role=%s tokenVersion=%d", u.Role, u.TokenVersion)
	}
}

func TestSetUserActive(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()
	seedUser(t, repo, &model.User{Username: "u", Role: "editor", IsActive: true, TokenVersion: 5})

	if err := setUserActive(ctx, repo, "u", false); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	u, _ := repo.GetByUsername(ctx, "u")
	if u.IsActive || u.TokenVersion != 6 {
		t.Fatalf("active=%v tokenVersion=%d (deactivation should bump version)", u.IsActive, u.TokenVersion)
	}

	if err := setUserActive(ctx, repo, "u", true); err != nil {
		t.Fatalf("activate: %v", err)
	}
	u, _ = repo.GetByUsername(ctx, "u")
	if !u.IsActive || u.TokenVersion != 6 {
		t.Fatalf("activation should not bump version: active=%v tokenVersion=%d", u.IsActive, u.TokenVersion)
	}
}

func TestLastSuperAdminGuard(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()
	seedUser(t, repo, &model.User{Username: "root", Role: "super-admin", IsActive: true})

	// The only super-admin cannot be demoted, deactivated, or deleted.
	if err := setUserRole(ctx, repo, "root", "viewer"); err == nil {
		t.Fatal("expected guard to block demoting the last super-admin")
	}
	if err := setUserActive(ctx, repo, "root", false); err == nil {
		t.Fatal("expected guard to block deactivating the last super-admin")
	}
	if err := deleteUserByName(ctx, repo, "root"); err == nil {
		t.Fatal("expected guard to block deleting the last super-admin")
	}

	// With a second super-admin present, the operation is allowed.
	seedUser(t, repo, &model.User{Username: "root2", Role: "super-admin", IsActive: true})
	if err := deleteUserByName(ctx, repo, "root"); err != nil {
		t.Fatalf("delete should succeed with a backup super-admin: %v", err)
	}
	if u, _ := repo.GetByUsername(ctx, "root"); u != nil {
		t.Fatal("user root should be deleted")
	}
}
