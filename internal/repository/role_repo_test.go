package repository

import (
	"context"
	"testing"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRoleRepo(t *testing.T) RoleRepo {
	t.Helper()
	db := setupTestDB(t)
	return NewRoleRepo(db)
}

func TestRoleRepo_GetByName(t *testing.T) {
	repo := newTestRoleRepo(t)
	ctx := context.Background()

	// System roles are seeded by setupTestDB.
	role, err := repo.GetByName(ctx, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)
	assert.True(t, role.IsSystem)
	assert.NotEmpty(t, role.Permissions)
}

func TestRoleRepo_GetByName_NotFound(t *testing.T) {
	repo := newTestRoleRepo(t)
	ctx := context.Background()

	_, err := repo.GetByName(ctx, "nonexistent")
	require.Error(t, err)
}

func TestRoleRepo_CreateAndGetByID(t *testing.T) {
	repo := newTestRoleRepo(t)
	ctx := context.Background()

	role := &model.Role{
		Name:        "custom-role",
		DisplayName: "Custom Role",
		Permissions: `["pods:list","services:get"]`,
	}
	err := repo.Create(ctx, role)
	require.NoError(t, err)
	assert.NotZero(t, role.ID)

	got, err := repo.GetByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, "custom-role", got.Name)
	assert.Equal(t, "Custom Role", got.DisplayName)
}

func TestRoleRepo_Update(t *testing.T) {
	repo := newTestRoleRepo(t)
	ctx := context.Background()

	role := &model.Role{
		Name:        "update-role",
		DisplayName: "Old Name",
		Permissions: `["pods:list"]`,
	}
	require.NoError(t, repo.Create(ctx, role))

	role.DisplayName = "New Name"
	role.Permissions = `["*:*"]`
	require.NoError(t, repo.Update(ctx, role))

	got, err := repo.GetByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", got.DisplayName)
	assert.Equal(t, `["*:*"]`, got.Permissions)
}

func TestRoleRepo_Delete(t *testing.T) {
	repo := newTestRoleRepo(t)
	ctx := context.Background()

	role := &model.Role{
		Name:        "delete-role",
		DisplayName: "To Delete",
		Permissions: `[]`,
	}
	require.NoError(t, repo.Create(ctx, role))

	require.NoError(t, repo.Delete(ctx, role.ID))

	_, err := repo.GetByID(ctx, role.ID)
	require.Error(t, err)
}

func TestRoleRepo_List(t *testing.T) {
	repo := newTestRoleRepo(t)
	ctx := context.Background()

	roles, err := repo.List(ctx)
	require.NoError(t, err)
	// 4 system roles seeded by default.
	assert.GreaterOrEqual(t, len(roles), 4)
}
