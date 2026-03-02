package repository

import (
	"context"
	"testing"

	"github.com/kubevision/kubevision/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClusterRepo creates a fresh in-memory DB and returns a ClusterRepo.
func newTestClusterRepo(t *testing.T) ClusterRepo {
	t.Helper()
	db := setupTestDB(t)
	return NewClusterRepo(db)
}

func TestClusterRepo_CreateAndGetByID(t *testing.T) {
	repo := newTestClusterRepo(t)
	ctx := context.Background()

	cluster := &model.Cluster{
		Name:        "test-cluster",
		DisplayName: "Test Cluster",
		APIServer:   "https://10.0.0.1:6443",
		AuthType:    "kubeconfig",
		Status:      "healthy",
	}
	err := repo.Create(ctx, cluster)
	require.NoError(t, err)
	assert.NotZero(t, cluster.ID, "cluster ID should be assigned after create")

	got, err := repo.GetByID(ctx, cluster.ID)
	require.NoError(t, err)
	assert.Equal(t, cluster.ID, got.ID)
	assert.Equal(t, "test-cluster", got.Name)
	assert.Equal(t, "Test Cluster", got.DisplayName)
	assert.Equal(t, "https://10.0.0.1:6443", got.APIServer)
	assert.Equal(t, "kubeconfig", got.AuthType)
	assert.Equal(t, "healthy", got.Status)
}

func TestClusterRepo_GetByID_NonExistent(t *testing.T) {
	repo := newTestClusterRepo(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	require.Error(t, err, "should return error for non-existent cluster ID")
}

func TestClusterRepo_GetByName(t *testing.T) {
	repo := newTestClusterRepo(t)
	ctx := context.Background()

	cluster := &model.Cluster{
		Name:     "prod-cluster",
		AuthType: "in-cluster",
		Status:   "healthy",
	}
	err := repo.Create(ctx, cluster)
	require.NoError(t, err)

	got, err := repo.GetByName(ctx, "prod-cluster")
	require.NoError(t, err)
	assert.Equal(t, cluster.ID, got.ID)
	assert.Equal(t, "prod-cluster", got.Name)
	assert.Equal(t, "in-cluster", got.AuthType)
}

func TestClusterRepo_GetByName_NonExistent(t *testing.T) {
	repo := newTestClusterRepo(t)
	ctx := context.Background()

	_, err := repo.GetByName(ctx, "nonexistent-cluster")
	require.Error(t, err, "should return error for non-existent cluster name")
}

func TestClusterRepo_List(t *testing.T) {
	repo := newTestClusterRepo(t)
	ctx := context.Background()

	// Initially there are no clusters.
	clusters, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, clusters, "initially no clusters should exist")

	// Create several clusters.
	for _, name := range []string{"cluster-a", "cluster-b", "cluster-c"} {
		err := repo.Create(ctx, &model.Cluster{
			Name:     name,
			AuthType: "kubeconfig",
			Status:   "unknown",
		})
		require.NoError(t, err)
	}

	clusters, err = repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, clusters, 3, "should list all 3 clusters")

	names := make(map[string]bool)
	for _, c := range clusters {
		names[c.Name] = true
	}
	assert.True(t, names["cluster-a"])
	assert.True(t, names["cluster-b"])
	assert.True(t, names["cluster-c"])
}

func TestClusterRepo_Update(t *testing.T) {
	repo := newTestClusterRepo(t)
	ctx := context.Background()

	cluster := &model.Cluster{
		Name:        "update-cluster",
		DisplayName: "Old Name",
		APIServer:   "https://old-server:6443",
		AuthType:    "kubeconfig",
		Status:      "unknown",
	}
	err := repo.Create(ctx, cluster)
	require.NoError(t, err)

	// Update fields.
	cluster.DisplayName = "New Display Name"
	cluster.APIServer = "https://new-server:6443"
	cluster.Status = "healthy"
	cluster.Version = "v1.28.0"
	err = repo.Update(ctx, cluster)
	require.NoError(t, err)

	// Verify the changes persisted.
	got, err := repo.GetByID(ctx, cluster.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Display Name", got.DisplayName)
	assert.Equal(t, "https://new-server:6443", got.APIServer)
	assert.Equal(t, "healthy", got.Status)
	assert.Equal(t, "v1.28.0", got.Version)
	assert.Equal(t, "update-cluster", got.Name, "name should not change")
}

func TestClusterRepo_Delete_SoftDeletes(t *testing.T) {
	repo := newTestClusterRepo(t)
	ctx := context.Background()

	cluster := &model.Cluster{
		Name:     "deletable-cluster",
		AuthType: "kubeconfig",
		Status:   "healthy",
	}
	err := repo.Create(ctx, cluster)
	require.NoError(t, err)
	clusterID := cluster.ID

	// Delete.
	err = repo.Delete(ctx, clusterID)
	require.NoError(t, err)

	// GetByID should not find the soft-deleted cluster.
	_, err = repo.GetByID(ctx, clusterID)
	require.Error(t, err, "soft-deleted cluster should not be found by GetByID")

	// GetByName should also not find it.
	_, err = repo.GetByName(ctx, "deletable-cluster")
	require.Error(t, err, "soft-deleted cluster should not be found by GetByName")
}

func TestClusterRepo_Delete_ExcludedFromList(t *testing.T) {
	repo := newTestClusterRepo(t)
	ctx := context.Background()

	// Create two clusters.
	c1 := &model.Cluster{Name: "keep-me", AuthType: "kubeconfig", Status: "healthy"}
	c2 := &model.Cluster{Name: "remove-me", AuthType: "kubeconfig", Status: "healthy"}
	require.NoError(t, repo.Create(ctx, c1))
	require.NoError(t, repo.Create(ctx, c2))

	// Delete one.
	require.NoError(t, repo.Delete(ctx, c2.ID))

	clusters, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, clusters, 1, "list should exclude soft-deleted cluster")
	assert.Equal(t, "keep-me", clusters[0].Name)
}

func TestClusterRepo_CreateDuplicateName(t *testing.T) {
	repo := newTestClusterRepo(t)
	ctx := context.Background()

	c1 := &model.Cluster{Name: "unique-name", AuthType: "kubeconfig", Status: "healthy"}
	err := repo.Create(ctx, c1)
	require.NoError(t, err)

	c2 := &model.Cluster{Name: "unique-name", AuthType: "kubeconfig", Status: "healthy"}
	err = repo.Create(ctx, c2)
	require.Error(t, err, "creating a cluster with duplicate name should fail due to unique index")
}
