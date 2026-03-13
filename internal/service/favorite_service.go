package service

import (
	"context"
	"time"

	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/repository"
)

// FavoriteResponse is the API response for a single favorite entry.
type FavoriteResponse struct {
	ID           uint      `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	UserID       uint      `json:"userId"`
	ClusterID    string    `json:"clusterId"`
	Namespace    string    `json:"namespace"`
	ResourceType string    `json:"resourceType"`
	ResourceName string    `json:"resourceName"`
	DisplayName  string    `json:"displayName"`
	SortOrder    int       `json:"sortOrder"`
}

// AddFavoriteRequest carries the data needed to create a new favorite.
type AddFavoriteRequest struct {
	ClusterID    string `json:"clusterId"    binding:"required"`
	Namespace    string `json:"namespace"`
	ResourceType string `json:"resourceType" binding:"required"`
	ResourceName string `json:"resourceName" binding:"required"`
	DisplayName  string `json:"displayName"`
}

// ReorderFavoritesRequest carries the ordered list of favorite IDs to apply.
type ReorderFavoritesRequest struct {
	OrderedIDs []uint `json:"orderedIds" binding:"required"`
}

// ToggleFavoriteResponse is returned by the toggle endpoint.
type ToggleFavoriteResponse struct {
	Favorited bool              `json:"favorited"`
	Favorite  *FavoriteResponse `json:"favorite,omitempty"`
}

// CheckFavoriteResponse reports whether a given resource is favorited.
type CheckFavoriteResponse struct {
	Favorited bool              `json:"favorited"`
	Favorite  *FavoriteResponse `json:"favorite,omitempty"`
}

// FavoriteService encapsulates business logic for the favorites/bookmarks system.
type FavoriteService struct {
	repo repository.FavoriteRepo
}

// NewFavoriteService creates a new FavoriteService.
func NewFavoriteService(repo repository.FavoriteRepo) *FavoriteService {
	return &FavoriteService{repo: repo}
}

// ListFavorites returns all favorites for the given user, ordered by SortOrder.
func (s *FavoriteService) ListFavorites(ctx context.Context, userID uint) ([]FavoriteResponse, error) {
	favs, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to list favorites")
	}

	result := make([]FavoriteResponse, len(favs))
	for i := range favs {
		result[i] = toFavoriteResponse(&favs[i])
	}
	return result, nil
}

// AddFavorite creates a new favorite for the given user. It returns a conflict
// error if the same resource is already favorited by this user.
func (s *FavoriteService) AddFavorite(ctx context.Context, userID uint, req *AddFavoriteRequest) (*FavoriteResponse, error) {
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.ResourceName
	}

	// Check for duplicates.
	existing, err := s.repo.GetByUserAndResource(ctx, userID, req.ClusterID, req.ResourceType, req.ResourceName, req.Namespace)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to check for existing favorite")
	}
	if existing != nil {
		return nil, bizerr.New(bizerr.CodeConflict, "resource is already favorited")
	}

	// Determine next sort order (append at end).
	all, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to determine sort order")
	}
	sortOrder := len(all)

	fav := &model.Favorite{
		UserID:       userID,
		ClusterID:    req.ClusterID,
		Namespace:    req.Namespace,
		ResourceType: req.ResourceType,
		ResourceName: req.ResourceName,
		DisplayName:  displayName,
		SortOrder:    sortOrder,
	}

	if err := s.repo.Create(ctx, fav); err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to create favorite")
	}

	resp := toFavoriteResponse(fav)
	return &resp, nil
}

// RemoveFavorite deletes a favorite by ID, enforcing ownership by userID.
func (s *FavoriteService) RemoveFavorite(ctx context.Context, userID uint, favoriteID uint) error {
	// Load the record to verify ownership.
	favs, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to load favorites")
	}

	found := false
	for _, f := range favs {
		if f.ID == favoriteID {
			found = true
			break
		}
	}
	if !found {
		return bizerr.New(bizerr.CodeNotFound, "favorite not found")
	}

	if err := s.repo.Delete(ctx, favoriteID); err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to delete favorite")
	}
	return nil
}

// ToggleFavorite adds the resource if it is not already favorited, or removes
// it if it is. Returns a ToggleFavoriteResponse describing the final state.
func (s *FavoriteService) ToggleFavorite(ctx context.Context, userID uint, req *AddFavoriteRequest) (*ToggleFavoriteResponse, error) {
	existing, err := s.repo.GetByUserAndResource(ctx, userID, req.ClusterID, req.ResourceType, req.ResourceName, req.Namespace)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to check for existing favorite")
	}

	if existing != nil {
		// Already favorited — remove it.
		if err := s.repo.Delete(ctx, existing.ID); err != nil {
			return nil, bizerr.New(bizerr.CodeInternal, "failed to remove favorite")
		}
		return &ToggleFavoriteResponse{Favorited: false}, nil
	}

	// Not yet favorited — add it.
	added, err := s.AddFavorite(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	return &ToggleFavoriteResponse{Favorited: true, Favorite: added}, nil
}

// ReorderFavorites updates the SortOrder of each favorite according to the
// position in orderedIDs. All IDs must belong to the given user.
func (s *FavoriteService) ReorderFavorites(ctx context.Context, userID uint, orderedIDs []uint) error {
	// Load user's favorites to validate ownership of all IDs.
	favs, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to load favorites")
	}

	ownedIDs := make(map[uint]bool, len(favs))
	for _, f := range favs {
		ownedIDs[f.ID] = true
	}

	for _, id := range orderedIDs {
		if !ownedIDs[id] {
			return bizerr.New(bizerr.CodeNotFound, "one or more favorites not found")
		}
	}

	for i, id := range orderedIDs {
		if err := s.repo.UpdateSortOrder(ctx, id, i); err != nil {
			return bizerr.New(bizerr.CodeInternal, "failed to update sort order")
		}
	}
	return nil
}

// CheckFavorite reports whether a resource is already favorited by the user.
func (s *FavoriteService) CheckFavorite(ctx context.Context, userID uint, clusterID, resourceType, resourceName, namespace string) (*CheckFavoriteResponse, error) {
	fav, err := s.repo.GetByUserAndResource(ctx, userID, clusterID, resourceType, resourceName, namespace)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to check favorite status")
	}
	if fav == nil {
		return &CheckFavoriteResponse{Favorited: false}, nil
	}
	resp := toFavoriteResponse(fav)
	return &CheckFavoriteResponse{Favorited: true, Favorite: &resp}, nil
}

// toFavoriteResponse converts a model.Favorite to its API response shape.
func toFavoriteResponse(f *model.Favorite) FavoriteResponse {
	return FavoriteResponse{
		ID:           f.ID,
		CreatedAt:    f.CreatedAt,
		UserID:       f.UserID,
		ClusterID:    f.ClusterID,
		Namespace:    f.Namespace,
		ResourceType: f.ResourceType,
		ResourceName: f.ResourceName,
		DisplayName:  f.DisplayName,
		SortOrder:    f.SortOrder,
	}
}
