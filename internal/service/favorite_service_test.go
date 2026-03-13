package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
)

// ---------------------------------------------------------------------------
// In-memory mock for repository.FavoriteRepo
// ---------------------------------------------------------------------------

type mockFavoriteRepo struct {
	favorites map[uint]*model.Favorite
	nextID    uint
	createErr error
	deleteErr error
	listErr   error
	updateErr error
	getByErr  error
	// When non-nil, getByResult is returned instead of a lookup result.
	getByResult *model.Favorite
}

func newMockFavoriteRepo() *mockFavoriteRepo {
	return &mockFavoriteRepo{
		favorites: make(map[uint]*model.Favorite),
		nextID:    1,
	}
}

func (m *mockFavoriteRepo) Create(_ context.Context, fav *model.Favorite) error {
	if m.createErr != nil {
		return m.createErr
	}
	fav.ID = m.nextID
	fav.CreatedAt = time.Now()
	m.nextID++
	// Store a copy.
	cp := *fav
	m.favorites[fav.ID] = &cp
	return nil
}

func (m *mockFavoriteRepo) Delete(_ context.Context, id uint) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.favorites, id)
	return nil
}

func (m *mockFavoriteRepo) ListByUser(_ context.Context, userID uint) ([]model.Favorite, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []model.Favorite
	for _, f := range m.favorites {
		if f.UserID == userID {
			result = append(result, *f)
		}
	}
	return result, nil
}

func (m *mockFavoriteRepo) UpdateSortOrder(_ context.Context, id uint, sortOrder int) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if f, ok := m.favorites[id]; ok {
		f.SortOrder = sortOrder
	}
	return nil
}

func (m *mockFavoriteRepo) GetByUserAndResource(
	_ context.Context,
	userID uint,
	clusterID, resourceType, resourceName, namespace string,
) (*model.Favorite, error) {
	if m.getByErr != nil {
		return nil, m.getByErr
	}
	// If the test has wired an explicit result, return it.
	if m.getByResult != nil {
		return m.getByResult, nil
	}
	for _, f := range m.favorites {
		if f.UserID == userID &&
			f.ClusterID == clusterID &&
			f.ResourceType == resourceType &&
			f.ResourceName == resourceName &&
			f.Namespace == namespace {
			cp := *f
			return &cp, nil
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newFavoriteSvc(repo *mockFavoriteRepo) *FavoriteService {
	return NewFavoriteService(repo)
}

func defaultAddReq() *AddFavoriteRequest {
	return &AddFavoriteRequest{
		ClusterID:    "cluster-a",
		Namespace:    "default",
		ResourceType: "deployments",
		ResourceName: "nginx",
		DisplayName:  "Nginx Deployment",
	}
}

// ---------------------------------------------------------------------------
// Tests: ListFavorites
// ---------------------------------------------------------------------------

func TestFavoriteService_ListFavorites(t *testing.T) {
	t.Run("returns empty slice when user has no favorites", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		resp, err := svc.ListFavorites(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp) != 0 {
			t.Errorf("expected 0 favorites, got %d", len(resp))
		}
	})

	t.Run("returns only favorites belonging to the requested user", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		// Add favorites for user 1.
		_ = repo.Create(context.Background(), &model.Favorite{UserID: 1, ClusterID: "c1", ResourceType: "pods", ResourceName: "pod-a"})
		_ = repo.Create(context.Background(), &model.Favorite{UserID: 1, ClusterID: "c1", ResourceType: "pods", ResourceName: "pod-b"})
		// Add favorite for user 2 — should not appear.
		_ = repo.Create(context.Background(), &model.Favorite{UserID: 2, ClusterID: "c1", ResourceType: "pods", ResourceName: "pod-c"})

		resp, err := svc.ListFavorites(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp) != 2 {
			t.Errorf("expected 2 favorites for user 1, got %d", len(resp))
		}
		for _, r := range resp {
			if r.UserID != 1 {
				t.Errorf("found favorite with UserID=%d, expected 1", r.UserID)
			}
		}
	})

	t.Run("propagates repo error as internal biz error", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		repo.listErr = errors.New("db failure")
		svc := newFavoriteSvc(repo)

		_, err := svc.ListFavorites(context.Background(), 1)
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeInternal {
			t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
		}
	})

	t.Run("response fields are correctly mapped from model", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		_ = repo.Create(context.Background(), &model.Favorite{
			UserID:       42,
			ClusterID:    "prod",
			Namespace:    "monitoring",
			ResourceType: "deployments",
			ResourceName: "prometheus",
			DisplayName:  "Prometheus",
			SortOrder:    5,
		})

		resp, err := svc.ListFavorites(context.Background(), 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp) != 1 {
			t.Fatalf("expected 1 favorite, got %d", len(resp))
		}
		r := resp[0]
		if r.ClusterID != "prod" {
			t.Errorf("ClusterID = %q, want %q", r.ClusterID, "prod")
		}
		if r.Namespace != "monitoring" {
			t.Errorf("Namespace = %q, want %q", r.Namespace, "monitoring")
		}
		if r.ResourceType != "deployments" {
			t.Errorf("ResourceType = %q, want %q", r.ResourceType, "deployments")
		}
		if r.ResourceName != "prometheus" {
			t.Errorf("ResourceName = %q, want %q", r.ResourceName, "prometheus")
		}
		if r.DisplayName != "Prometheus" {
			t.Errorf("DisplayName = %q, want %q", r.DisplayName, "Prometheus")
		}
		if r.SortOrder != 5 {
			t.Errorf("SortOrder = %d, want 5", r.SortOrder)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: AddFavorite
// ---------------------------------------------------------------------------

func TestFavoriteService_AddFavorite(t *testing.T) {
	t.Run("successfully creates a new favorite", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		req := defaultAddReq()
		resp, err := svc.AddFavorite(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if resp.ResourceName != "nginx" {
			t.Errorf("ResourceName = %q, want %q", resp.ResourceName, "nginx")
		}
		if resp.DisplayName != "Nginx Deployment" {
			t.Errorf("DisplayName = %q, want %q", resp.DisplayName, "Nginx Deployment")
		}
	})

	t.Run("uses resource name as display name when display name is empty", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		req := &AddFavoriteRequest{
			ClusterID:    "cluster-a",
			Namespace:    "default",
			ResourceType: "pods",
			ResourceName: "my-pod",
			DisplayName:  "", // empty — should fall back to ResourceName
		}
		resp, err := svc.AddFavorite(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.DisplayName != "my-pod" {
			t.Errorf("DisplayName = %q, want %q", resp.DisplayName, "my-pod")
		}
	})

	t.Run("sort order is set to position at end of list", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		// Pre-populate two existing favorites for the same user.
		_ = repo.Create(context.Background(), &model.Favorite{UserID: 1, ClusterID: "c", ResourceType: "pods", ResourceName: "a"})
		_ = repo.Create(context.Background(), &model.Favorite{UserID: 1, ClusterID: "c", ResourceType: "pods", ResourceName: "b"})

		req := &AddFavoriteRequest{
			ClusterID:    "c",
			Namespace:    "default",
			ResourceType: "pods",
			ResourceName: "c-pod",
		}
		resp, err := svc.AddFavorite(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Two existing favorites means index 2 (zero-based).
		if resp.SortOrder != 2 {
			t.Errorf("SortOrder = %d, want 2", resp.SortOrder)
		}
	})

	t.Run("returns conflict error when resource is already favorited", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		req := defaultAddReq()
		// First add should succeed.
		_, err := svc.AddFavorite(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("first AddFavorite failed: %v", err)
		}
		// Second add for the same resource should return conflict.
		_, err = svc.AddFavorite(context.Background(), 1, req)
		if err == nil {
			t.Fatal("expected conflict error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeConflict {
			t.Errorf("expected code %d, got %d", bizerr.CodeConflict, bizErr.Code)
		}
	})

	t.Run("GetByUserAndResource error returns internal error", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		repo.getByErr = errors.New("db broken")
		svc := newFavoriteSvc(repo)

		_, err := svc.AddFavorite(context.Background(), 1, defaultAddReq())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeInternal {
			t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
		}
	})

	t.Run("ListByUser error during sort order determination returns internal error", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)
		// getBy returns nil (no duplicate), but ListByUser fails.
		repo.listErr = errors.New("db failure during list")

		_, err := svc.AddFavorite(context.Background(), 1, defaultAddReq())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeInternal {
			t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
		}
	})

	t.Run("repo Create error returns internal error", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		repo.createErr = errors.New("insert failed")
		svc := newFavoriteSvc(repo)

		_, err := svc.AddFavorite(context.Background(), 1, defaultAddReq())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeInternal {
			t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: RemoveFavorite
// ---------------------------------------------------------------------------

func TestFavoriteService_RemoveFavorite(t *testing.T) {
	t.Run("successfully removes an owned favorite", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		_ = repo.Create(context.Background(), &model.Favorite{
			UserID: 1, ClusterID: "c", ResourceType: "pods", ResourceName: "pod-x",
		})
		// ID is 1 (first insert).
		err := svc.RemoveFavorite(context.Background(), 1, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The favorite should no longer be in the store.
		if _, exists := repo.favorites[1]; exists {
			t.Error("expected favorite to be deleted from repo")
		}
	})

	t.Run("returns not-found error for unknown favorite ID", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		err := svc.RemoveFavorite(context.Background(), 1, 999)
		if err == nil {
			t.Fatal("expected error for non-existent favorite")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeNotFound {
			t.Errorf("expected code %d, got %d", bizerr.CodeNotFound, bizErr.Code)
		}
	})

	t.Run("returns not-found error when favorite belongs to a different user", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		// Favorite is owned by user 2.
		_ = repo.Create(context.Background(), &model.Favorite{
			UserID: 2, ClusterID: "c", ResourceType: "pods", ResourceName: "pod-y",
		})

		// User 1 tries to remove user 2's favorite.
		err := svc.RemoveFavorite(context.Background(), 1, 1)
		if err == nil {
			t.Fatal("expected not-found error for wrong owner")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeNotFound {
			t.Errorf("expected code %d, got %d", bizerr.CodeNotFound, bizErr.Code)
		}
	})

	t.Run("ListByUser error returns internal error", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		repo.listErr = errors.New("db down")
		svc := newFavoriteSvc(repo)

		err := svc.RemoveFavorite(context.Background(), 1, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeInternal {
			t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
		}
	})

	t.Run("Delete error returns internal error", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		_ = repo.Create(context.Background(), &model.Favorite{
			UserID: 1, ClusterID: "c", ResourceType: "pods", ResourceName: "pod-z",
		})
		repo.deleteErr = errors.New("delete failed")

		err := svc.RemoveFavorite(context.Background(), 1, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeInternal {
			t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: ToggleFavorite
// ---------------------------------------------------------------------------

func TestFavoriteService_ToggleFavorite(t *testing.T) {
	t.Run("adds favorite when not yet favorited", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		resp, err := svc.ToggleFavorite(context.Background(), 1, defaultAddReq())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Favorited {
			t.Error("expected Favorited=true after first toggle")
		}
		if resp.Favorite == nil {
			t.Error("expected non-nil Favorite in response when adding")
		}
	})

	t.Run("removes favorite when already favorited", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		req := defaultAddReq()
		// Add first.
		_, err := svc.ToggleFavorite(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("first toggle failed: %v", err)
		}
		// Toggle again — should remove.
		resp, err := svc.ToggleFavorite(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("second toggle failed: %v", err)
		}
		if resp.Favorited {
			t.Error("expected Favorited=false after second toggle")
		}
		if resp.Favorite != nil {
			t.Error("expected nil Favorite in response when removing")
		}
	})

	t.Run("GetByUserAndResource error returns internal biz error", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		repo.getByErr = errors.New("db failure")
		svc := newFavoriteSvc(repo)

		_, err := svc.ToggleFavorite(context.Background(), 1, defaultAddReq())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeInternal {
			t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
		}
	})

	t.Run("Delete error during unfavorite returns internal error", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		req := defaultAddReq()
		// Pre-add so toggle will attempt to remove.
		_, err := svc.ToggleFavorite(context.Background(), 1, req)
		if err != nil {
			t.Fatalf("first toggle failed: %v", err)
		}
		// Now make Delete fail.
		repo.deleteErr = errors.New("delete failure")

		_, err = svc.ToggleFavorite(context.Background(), 1, req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeInternal {
			t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: ReorderFavorites
// ---------------------------------------------------------------------------

func TestFavoriteService_ReorderFavorites(t *testing.T) {
	t.Run("successfully reorders favorites", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		_ = repo.Create(context.Background(), &model.Favorite{UserID: 1, ClusterID: "c", ResourceType: "pods", ResourceName: "a"})
		_ = repo.Create(context.Background(), &model.Favorite{UserID: 1, ClusterID: "c", ResourceType: "pods", ResourceName: "b"})
		_ = repo.Create(context.Background(), &model.Favorite{UserID: 1, ClusterID: "c", ResourceType: "pods", ResourceName: "c-pod"})
		// IDs are 1, 2, 3.

		err := svc.ReorderFavorites(context.Background(), 1, []uint{3, 1, 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify sort orders were updated.
		if repo.favorites[3].SortOrder != 0 {
			t.Errorf("ID=3 SortOrder = %d, want 0", repo.favorites[3].SortOrder)
		}
		if repo.favorites[1].SortOrder != 1 {
			t.Errorf("ID=1 SortOrder = %d, want 1", repo.favorites[1].SortOrder)
		}
		if repo.favorites[2].SortOrder != 2 {
			t.Errorf("ID=2 SortOrder = %d, want 2", repo.favorites[2].SortOrder)
		}
	})

	t.Run("returns not-found error when ID does not belong to user", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		_ = repo.Create(context.Background(), &model.Favorite{UserID: 1, ClusterID: "c", ResourceType: "pods", ResourceName: "a"})
		// Attempt to reorder with a foreign ID (999).
		err := svc.ReorderFavorites(context.Background(), 1, []uint{1, 999})
		if err == nil {
			t.Fatal("expected not-found error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeNotFound {
			t.Errorf("expected code %d, got %d", bizerr.CodeNotFound, bizErr.Code)
		}
	})

	t.Run("returns not-found error for other user's favorite", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		_ = repo.Create(context.Background(), &model.Favorite{UserID: 1, ClusterID: "c", ResourceType: "pods", ResourceName: "a"})
		_ = repo.Create(context.Background(), &model.Favorite{UserID: 2, ClusterID: "c", ResourceType: "pods", ResourceName: "b"})

		// User 1 tries to reorder with ID 2 which belongs to user 2.
		err := svc.ReorderFavorites(context.Background(), 1, []uint{1, 2})
		if err == nil {
			t.Fatal("expected not-found error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeNotFound {
			t.Errorf("expected code %d, got %d", bizerr.CodeNotFound, bizErr.Code)
		}
	})

	t.Run("ListByUser error returns internal error", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		repo.listErr = errors.New("db failure")
		svc := newFavoriteSvc(repo)

		err := svc.ReorderFavorites(context.Background(), 1, []uint{1})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeInternal {
			t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
		}
	})

	t.Run("UpdateSortOrder error returns internal error", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		_ = repo.Create(context.Background(), &model.Favorite{UserID: 1, ClusterID: "c", ResourceType: "pods", ResourceName: "a"})
		repo.updateErr = errors.New("update failed")

		err := svc.ReorderFavorites(context.Background(), 1, []uint{1})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeInternal {
			t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: CheckFavorite
// ---------------------------------------------------------------------------

func TestFavoriteService_CheckFavorite(t *testing.T) {
	t.Run("returns Favorited=false when resource is not favorited", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		resp, err := svc.CheckFavorite(context.Background(), 1, "cluster-a", "pods", "my-pod", "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Favorited {
			t.Error("expected Favorited=false")
		}
		if resp.Favorite != nil {
			t.Error("expected nil Favorite when not favorited")
		}
	})

	t.Run("returns Favorited=true with detail when resource is favorited", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		svc := newFavoriteSvc(repo)

		_ = repo.Create(context.Background(), &model.Favorite{
			UserID:       1,
			ClusterID:    "cluster-a",
			Namespace:    "default",
			ResourceType: "pods",
			ResourceName: "my-pod",
			DisplayName:  "my-pod",
		})

		resp, err := svc.CheckFavorite(context.Background(), 1, "cluster-a", "pods", "my-pod", "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Favorited {
			t.Error("expected Favorited=true")
		}
		if resp.Favorite == nil {
			t.Fatal("expected non-nil Favorite when favorited")
		}
		if resp.Favorite.ResourceName != "my-pod" {
			t.Errorf("Favorite.ResourceName = %q, want %q", resp.Favorite.ResourceName, "my-pod")
		}
	})

	t.Run("repo error returns internal biz error", func(t *testing.T) {
		repo := newMockFavoriteRepo()
		repo.getByErr = errors.New("db error")
		svc := newFavoriteSvc(repo)

		_, err := svc.CheckFavorite(context.Background(), 1, "cluster-a", "pods", "my-pod", "default")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T", err)
		}
		if bizErr.Code != bizerr.CodeInternal {
			t.Errorf("expected code %d, got %d", bizerr.CodeInternal, bizErr.Code)
		}
	})
}
