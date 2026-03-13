package repository

import (
	"context"
	"testing"

	"github.com/gocronx/kubevision/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestFavoriteRepo creates a fresh in-memory DB and returns a FavoriteRepo.
func newTestFavoriteRepo(t *testing.T) FavoriteRepo {
	t.Helper()
	db := setupTestDB(t)
	return NewFavoriteRepo(db)
}

// seedFavorite is a convenience helper that inserts a Favorite and fails the
// test immediately if the insert fails.
func seedFavorite(t *testing.T, repo FavoriteRepo, fav *model.Favorite) {
	t.Helper()
	require.NoError(t, repo.Create(context.Background(), fav))
	require.NotZero(t, fav.ID, "favorite ID should be assigned after create")
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestFavoriteRepo_Create(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	fav := &model.Favorite{
		UserID:       1,
		ClusterID:    "cluster-a",
		Namespace:    "default",
		ResourceType: "pods",
		ResourceName: "my-pod",
		DisplayName:  "My Pod",
		SortOrder:    0,
	}
	err := repo.Create(context.Background(), fav)
	require.NoError(t, err)
	assert.NotZero(t, fav.ID, "ID should be populated after create")
}

func TestFavoriteRepo_Create_MultipleForSameUser(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	for i, name := range []string{"pod-a", "pod-b", "pod-c"} {
		fav := &model.Favorite{
			UserID:       10,
			ClusterID:    "cluster-x",
			Namespace:    "default",
			ResourceType: "pods",
			ResourceName: name,
			DisplayName:  name,
			SortOrder:    i,
		}
		require.NoError(t, repo.Create(context.Background(), fav))
		assert.NotZero(t, fav.ID)
	}
}

// Favorite records do not have a unique constraint across all fields by default;
// duplicates are allowed (the caller deduplicates via GetByUserAndResource).
func TestFavoriteRepo_Create_DuplicateAllowed(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	fav1 := &model.Favorite{
		UserID:       5,
		ClusterID:    "cluster-a",
		Namespace:    "ns",
		ResourceType: "deployments",
		ResourceName: "my-deploy",
	}
	err := repo.Create(context.Background(), fav1)
	require.NoError(t, err)

	// Insert the exact same logical resource again — the DB should accept it
	// because there is no unique index on (user_id, cluster_id, resource_type,
	// resource_name, namespace).
	fav2 := &model.Favorite{
		UserID:       5,
		ClusterID:    "cluster-a",
		Namespace:    "ns",
		ResourceType: "deployments",
		ResourceName: "my-deploy",
	}
	err = repo.Create(context.Background(), fav2)
	require.NoError(t, err)
	assert.NotEqual(t, fav1.ID, fav2.ID, "each insert should create a distinct row")
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestFavoriteRepo_Delete_ExistingRecord(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	fav := &model.Favorite{
		UserID:       1,
		ClusterID:    "cluster-a",
		Namespace:    "kube-system",
		ResourceType: "nodes",
		ResourceName: "node-1",
	}
	seedFavorite(t, repo, fav)
	favID := fav.ID

	err := repo.Delete(context.Background(), favID)
	require.NoError(t, err)

	// After deletion, ListByUser should no longer include this record.
	list, err := repo.ListByUser(context.Background(), fav.UserID)
	require.NoError(t, err)
	for _, f := range list {
		assert.NotEqual(t, favID, f.ID, "deleted favorite should not appear in list")
	}
}

func TestFavoriteRepo_Delete_NonExistentID_NoError(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	// GORM's Delete does not return an error for a record that does not exist
	// (the DELETE statement simply affects 0 rows). The repo matches this
	// behaviour.
	err := repo.Delete(context.Background(), 99999)
	require.NoError(t, err)
}

func TestFavoriteRepo_Delete_OnlyRemovesTargetRecord(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	fav1 := &model.Favorite{UserID: 2, ClusterID: "c1", Namespace: "ns", ResourceType: "pods", ResourceName: "pod-keep"}
	fav2 := &model.Favorite{UserID: 2, ClusterID: "c1", Namespace: "ns", ResourceType: "pods", ResourceName: "pod-delete"}
	seedFavorite(t, repo, fav1)
	seedFavorite(t, repo, fav2)

	require.NoError(t, repo.Delete(context.Background(), fav2.ID))

	list, err := repo.ListByUser(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, fav1.ID, list[0].ID)
}

// ---------------------------------------------------------------------------
// ListByUser
// ---------------------------------------------------------------------------

func TestFavoriteRepo_ListByUser_Empty(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	list, err := repo.ListByUser(context.Background(), 42)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestFavoriteRepo_ListByUser_ReturnsOnlyOwnersFavorites(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	// User 1 favorites.
	for _, name := range []string{"svc-a", "svc-b"} {
		seedFavorite(t, repo, &model.Favorite{
			UserID: 1, ClusterID: "c", Namespace: "ns", ResourceType: "services", ResourceName: name,
		})
	}
	// User 2 favorites.
	seedFavorite(t, repo, &model.Favorite{
		UserID: 2, ClusterID: "c", Namespace: "ns", ResourceType: "services", ResourceName: "svc-x",
	})

	user1List, err := repo.ListByUser(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, user1List, 2, "user 1 should have exactly 2 favorites")

	user2List, err := repo.ListByUser(context.Background(), 2)
	require.NoError(t, err)
	assert.Len(t, user2List, 1, "user 2 should have exactly 1 favorite")
}

func TestFavoriteRepo_ListByUser_SortedBySortOrderAsc(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	// Insert in reverse sort order.
	for i, name := range []string{"pod-c", "pod-a", "pod-b"} {
		sortOrder := (2 - i) * 10 // 20, 10, 0
		seedFavorite(t, repo, &model.Favorite{
			UserID: 3, ClusterID: "c", Namespace: "ns",
			ResourceType: "pods", ResourceName: name,
			SortOrder: sortOrder,
		})
	}

	list, err := repo.ListByUser(context.Background(), 3)
	require.NoError(t, err)
	require.Len(t, list, 3)

	// Expect ascending sort order: 0, 10, 20.
	assert.LessOrEqual(t, list[0].SortOrder, list[1].SortOrder)
	assert.LessOrEqual(t, list[1].SortOrder, list[2].SortOrder)
}

// ---------------------------------------------------------------------------
// GetByUserAndResource
// ---------------------------------------------------------------------------

func TestFavoriteRepo_GetByUserAndResource_Found(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	fav := &model.Favorite{
		UserID:       7,
		ClusterID:    "prod",
		Namespace:    "production",
		ResourceType: "deployments",
		ResourceName: "api-server",
		DisplayName:  "API Server Deployment",
	}
	seedFavorite(t, repo, fav)

	got, err := repo.GetByUserAndResource(
		context.Background(),
		7, "prod", "deployments", "api-server", "production",
	)
	require.NoError(t, err)
	require.NotNil(t, got, "should find the favorite")
	assert.Equal(t, fav.ID, got.ID)
	assert.Equal(t, "API Server Deployment", got.DisplayName)
}

func TestFavoriteRepo_GetByUserAndResource_NotFound_ReturnsNil(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	got, err := repo.GetByUserAndResource(
		context.Background(),
		99, "missing-cluster", "pods", "nonexistent-pod", "ns",
	)
	require.NoError(t, err, "GetByUserAndResource should return (nil, nil) when not found")
	assert.Nil(t, got, "result should be nil for non-existent favorite")
}

func TestFavoriteRepo_GetByUserAndResource_IsolatedByUser(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	// User 100 owns this favorite.
	seedFavorite(t, repo, &model.Favorite{
		UserID:       100,
		ClusterID:    "c",
		Namespace:    "ns",
		ResourceType: "pods",
		ResourceName: "shared-pod",
	})

	// User 200 should NOT find the same resource.
	got, err := repo.GetByUserAndResource(
		context.Background(),
		200, "c", "pods", "shared-pod", "ns",
	)
	require.NoError(t, err)
	assert.Nil(t, got, "user 200 should not find user 100's favorite")
}

func TestFavoriteRepo_GetByUserAndResource_MatchesAllFields(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	// Seed two favorites that differ only in one field each.
	seedFavorite(t, repo, &model.Favorite{
		UserID: 8, ClusterID: "c1", Namespace: "ns1", ResourceType: "pods", ResourceName: "pod-x",
	})
	seedFavorite(t, repo, &model.Favorite{
		UserID: 8, ClusterID: "c2", Namespace: "ns1", ResourceType: "pods", ResourceName: "pod-x",
	})

	// Query for cluster c1.
	got, err := repo.GetByUserAndResource(context.Background(), 8, "c1", "pods", "pod-x", "ns1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "c1", got.ClusterID)

	// Query for cluster c2.
	got2, err := repo.GetByUserAndResource(context.Background(), 8, "c2", "pods", "pod-x", "ns1")
	require.NoError(t, err)
	require.NotNil(t, got2)
	assert.Equal(t, "c2", got2.ClusterID)

	// Different namespace should not match.
	got3, err := repo.GetByUserAndResource(context.Background(), 8, "c1", "pods", "pod-x", "other-ns")
	require.NoError(t, err)
	assert.Nil(t, got3, "should not match a different namespace")
}

// ---------------------------------------------------------------------------
// UpdateSortOrder
// ---------------------------------------------------------------------------

func TestFavoriteRepo_UpdateSortOrder_ChangesValue(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	fav := &model.Favorite{
		UserID:       4,
		ClusterID:    "c",
		Namespace:    "ns",
		ResourceType: "pods",
		ResourceName: "reorder-me",
		SortOrder:    0,
	}
	seedFavorite(t, repo, fav)

	err := repo.UpdateSortOrder(context.Background(), fav.ID, 99)
	require.NoError(t, err)

	// Verify via list.
	list, err := repo.ListByUser(context.Background(), 4)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, 99, list[0].SortOrder)
}

func TestFavoriteRepo_UpdateSortOrder_NonExistentID_NoError(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	// GORM UPDATE with a WHERE clause that matches no rows is not an error.
	err := repo.UpdateSortOrder(context.Background(), 99999, 50)
	require.NoError(t, err)
}

func TestFavoriteRepo_UpdateSortOrder_MultipleItems_OnlyTargetChanges(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	fav1 := &model.Favorite{UserID: 6, ClusterID: "c", Namespace: "ns", ResourceType: "pods", ResourceName: "pod-1", SortOrder: 1}
	fav2 := &model.Favorite{UserID: 6, ClusterID: "c", Namespace: "ns", ResourceType: "pods", ResourceName: "pod-2", SortOrder: 2}
	seedFavorite(t, repo, fav1)
	seedFavorite(t, repo, fav2)

	// Update only fav1's sort order.
	require.NoError(t, repo.UpdateSortOrder(context.Background(), fav1.ID, 100))

	list, err := repo.ListByUser(context.Background(), 6)
	require.NoError(t, err)
	require.Len(t, list, 2)

	// Sort order 2 should come first in ascending order, then 100.
	assert.Equal(t, 2, list[0].SortOrder, "fav2 sort_order should be unchanged")
	assert.Equal(t, 100, list[1].SortOrder, "fav1 sort_order should be updated to 100")
}

// ---------------------------------------------------------------------------
// User isolation: end-to-end cross-user scenario
// ---------------------------------------------------------------------------

func TestFavoriteRepo_UserIsolation(t *testing.T) {
	repo := newTestFavoriteRepo(t)

	// Three users each with their own favorites.
	const userA, userB, userC uint = 101, 102, 103

	seedFavorite(t, repo, &model.Favorite{UserID: userA, ClusterID: "c", Namespace: "ns", ResourceType: "pods", ResourceName: "a-pod"})
	seedFavorite(t, repo, &model.Favorite{UserID: userA, ClusterID: "c", Namespace: "ns", ResourceType: "pods", ResourceName: "a-pod-2"})
	seedFavorite(t, repo, &model.Favorite{UserID: userB, ClusterID: "c", Namespace: "ns", ResourceType: "services", ResourceName: "b-svc"})
	seedFavorite(t, repo, &model.Favorite{UserID: userC, ClusterID: "c", Namespace: "ns", ResourceType: "deployments", ResourceName: "c-deploy"})

	listA, err := repo.ListByUser(context.Background(), userA)
	require.NoError(t, err)
	assert.Len(t, listA, 2)

	listB, err := repo.ListByUser(context.Background(), userB)
	require.NoError(t, err)
	assert.Len(t, listB, 1)

	listC, err := repo.ListByUser(context.Background(), userC)
	require.NoError(t, err)
	assert.Len(t, listC, 1)

	// After deleting userA's first favorite, userB and userC are unaffected.
	require.NoError(t, repo.Delete(context.Background(), listA[0].ID))

	listAAfter, err := repo.ListByUser(context.Background(), userA)
	require.NoError(t, err)
	assert.Len(t, listAAfter, 1)

	listBAfter, err := repo.ListByUser(context.Background(), userB)
	require.NoError(t, err)
	assert.Len(t, listBAfter, 1, "userB favorites should be unaffected")

	listCAfter, err := repo.ListByUser(context.Background(), userC)
	require.NoError(t, err)
	assert.Len(t, listCAfter, 1, "userC favorites should be unaffected")
}
