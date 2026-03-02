package repository

import (
	"context"
	"testing"

	"github.com/kubevision/kubevision/internal/auth"
	"github.com/kubevision/kubevision/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestUserRepo creates a fresh in-memory DB and returns a UserRepo.
// The DB already contains the seeded admin user from NewDB.
func newTestUserRepo(t *testing.T) UserRepo {
	t.Helper()
	db := setupTestDB(t)
	return NewUserRepo(db)
}

func TestUserRepo_CreateAndGetByID(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("testpass")
	require.NoError(t, err)

	user := &model.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: hash,
		Role:         "dev",
		IsActive:     true,
	}
	err = repo.Create(ctx, user)
	require.NoError(t, err)
	assert.NotZero(t, user.ID, "user ID should be assigned after create")

	// Retrieve by ID.
	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)
	assert.Equal(t, "testuser", got.Username)
	assert.Equal(t, "test@example.com", got.Email)
	assert.Equal(t, "dev", got.Role)
	assert.True(t, got.IsActive)
	assert.NotEmpty(t, got.PasswordHash)
}

func TestUserRepo_GetByUsername(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("pass123")
	require.NoError(t, err)

	user := &model.User{
		Username:     "lookup_user",
		PasswordHash: hash,
		Role:         "ops",
		IsActive:     true,
	}
	err = repo.Create(ctx, user)
	require.NoError(t, err)

	got, err := repo.GetByUsername(ctx, "lookup_user")
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)
	assert.Equal(t, "lookup_user", got.Username)
	assert.Equal(t, "ops", got.Role)
}

func TestUserRepo_GetByUsername_NonExistent(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	_, err := repo.GetByUsername(ctx, "nonexistent_user")
	require.Error(t, err, "should return error for non-existent username")
}

func TestUserRepo_GetByID_NonExistent(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	require.Error(t, err, "should return error for non-existent user ID")
}

func TestUserRepo_Update(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("original")
	require.NoError(t, err)

	user := &model.User{
		Username:     "update_user",
		Email:        "old@example.com",
		PasswordHash: hash,
		Role:         "dev",
		IsActive:     true,
	}
	err = repo.Create(ctx, user)
	require.NoError(t, err)

	// Modify fields.
	user.Email = "new@example.com"
	user.Role = "admin"
	err = repo.Update(ctx, user)
	require.NoError(t, err)

	// Verify the changes persisted.
	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", got.Email)
	assert.Equal(t, "admin", got.Role)
	assert.Equal(t, "update_user", got.Username, "username should not change")
}

func TestUserRepo_Delete_SoftDeletes(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("pass")
	require.NoError(t, err)

	user := &model.User{
		Username:     "delete_me",
		PasswordHash: hash,
		Role:         "readonly",
		IsActive:     true,
	}
	err = repo.Create(ctx, user)
	require.NoError(t, err)
	userID := user.ID

	// Delete the user.
	err = repo.Delete(ctx, userID)
	require.NoError(t, err)

	// GetByID should no longer find the soft-deleted user.
	_, err = repo.GetByID(ctx, userID)
	require.Error(t, err, "soft-deleted user should not be found by GetByID")

	// GetByUsername should also not find it.
	_, err = repo.GetByUsername(ctx, "delete_me")
	require.Error(t, err, "soft-deleted user should not be found by GetByUsername")
}

func TestUserRepo_List(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	// The seeded admin user already exists. Add more users.
	for _, name := range []string{"alice", "bob", "charlie"} {
		hash, err := auth.HashPassword("pass")
		require.NoError(t, err)
		err = repo.Create(ctx, &model.User{
			Username:     name,
			PasswordHash: hash,
			Role:         "dev",
			IsActive:     true,
		})
		require.NoError(t, err)
	}

	users, err := repo.List(ctx)
	require.NoError(t, err)
	// 1 admin + 3 test users = 4.
	assert.Len(t, users, 4, "should list all users including seeded admin")

	// Collect usernames.
	names := make(map[string]bool)
	for _, u := range users {
		names[u.Username] = true
	}
	assert.True(t, names["admin"], "admin should be in list")
	assert.True(t, names["alice"], "alice should be in list")
	assert.True(t, names["bob"], "bob should be in list")
	assert.True(t, names["charlie"], "charlie should be in list")
}

func TestUserRepo_List_ExcludesSoftDeleted(t *testing.T) {
	repo := newTestUserRepo(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("pass")
	require.NoError(t, err)

	user := &model.User{
		Username:     "to_be_deleted",
		PasswordHash: hash,
		Role:         "readonly",
		IsActive:     true,
	}
	err = repo.Create(ctx, user)
	require.NoError(t, err)

	// Count before delete.
	beforeList, err := repo.List(ctx)
	require.NoError(t, err)
	beforeCount := len(beforeList)

	// Soft delete.
	err = repo.Delete(ctx, user.ID)
	require.NoError(t, err)

	afterList, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, beforeCount-1, len(afterList), "list should exclude soft-deleted users")
}
