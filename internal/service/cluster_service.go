package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/kubevision/kubevision/internal/auth"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/kubernetes/cluster"
	"github.com/kubevision/kubevision/internal/kubernetes/informer"
	"github.com/kubevision/kubevision/internal/kubernetes/resource"
	"github.com/kubevision/kubevision/internal/model"
	"github.com/kubevision/kubevision/internal/repository"
)

// ClusterResponse is the API response for a cluster.
type ClusterResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	APIServer   string    `json:"apiServer"`
	AuthType    string    `json:"authType"`
	Status      string    `json:"status"`
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
}

// AddClusterRequest is the request body for adding a cluster.
type AddClusterRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"displayName"`
	AuthType    string `json:"authType" binding:"required"` // kubeconfig | in-cluster
	Kubeconfig  string `json:"kubeconfig"`
}

// ClusterService encapsulates business logic for Kubernetes cluster management.
type ClusterService struct {
	clusterRepo    repository.ClusterRepo
	clusterManager *cluster.Manager
	informerMgr    *informer.Manager
	registry       *resource.Registry
	logger         *zap.Logger
	encryptKey     string
}

// NewClusterService creates a new ClusterService.
func NewClusterService(
	clusterRepo repository.ClusterRepo,
	clusterManager *cluster.Manager,
	informerMgr *informer.Manager,
	registry *resource.Registry,
	logger *zap.Logger,
	encryptKey string,
) *ClusterService {
	return &ClusterService{
		clusterRepo:    clusterRepo,
		clusterManager: clusterManager,
		informerMgr:    informerMgr,
		registry:       registry,
		logger:         logger,
		encryptKey:     encryptKey,
	}
}

// Add registers a new cluster, establishes a K8s connection, and starts informers.
func (s *ClusterService) Add(ctx context.Context, req *AddClusterRequest) (*ClusterResponse, error) {
	// Check name uniqueness.
	if existing, _ := s.clusterRepo.GetByName(ctx, req.Name); existing != nil {
		return nil, bizerr.New(bizerr.CodeConflict, fmt.Sprintf("cluster %q already exists", req.Name))
	}

	// Register the cluster with the K8s cluster manager.
	clusterID := req.Name // use name as cluster ID for simplicity
	switch req.AuthType {
	case "kubeconfig":
		if req.Kubeconfig == "" {
			return nil, bizerr.New(bizerr.CodeParamInvalid, "kubeconfig is required for auth type 'kubeconfig'")
		}
		if err := s.clusterManager.Add(clusterID, []byte(req.Kubeconfig)); err != nil {
			return nil, bizerr.New(bizerr.CodeK8sUnavailable, fmt.Sprintf("failed to connect: %s", err.Error()))
		}
	case "in-cluster":
		if err := s.clusterManager.AddInCluster(clusterID); err != nil {
			return nil, bizerr.New(bizerr.CodeK8sUnavailable, fmt.Sprintf("failed to connect: %s", err.Error()))
		}
	default:
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("unsupported auth type: %s", req.AuthType))
	}

	// Encrypt kubeconfig before persisting.
	var kubeconfigEnc string
	if req.Kubeconfig != "" {
		encrypted, err := auth.Encrypt(req.Kubeconfig, s.encryptKey)
		if err != nil {
			s.clusterManager.Remove(clusterID)
			return nil, bizerr.New(bizerr.CodeInternal, "failed to encrypt kubeconfig")
		}
		kubeconfigEnc = encrypted
	}

	// Persist to database.
	m := &model.Cluster{
		Name:          req.Name,
		DisplayName:   req.DisplayName,
		AuthType:      req.AuthType,
		KubeconfigEnc: kubeconfigEnc,
		Status:        "healthy",
	}
	if err := s.clusterRepo.Create(ctx, m); err != nil {
		// Rollback K8s connection on DB failure.
		s.clusterManager.Remove(clusterID)
		return nil, bizerr.New(bizerr.CodeInternal, "failed to save cluster")
	}

	// Start informers for cached resources.
	dynClient, err := s.clusterManager.DynamicClient(clusterID)
	if err == nil {
		s.informerMgr.StartForCluster(clusterID, dynClient, s.registry.CachedGVRs(), 30*time.Minute)
	}

	s.logger.Info("cluster added", zap.String("name", req.Name), zap.Uint("id", m.ID))

	return toClusterResponse(m), nil
}

// List returns all registered clusters.
func (s *ClusterService) List(ctx context.Context) ([]ClusterResponse, error) {
	clusters, err := s.clusterRepo.List(ctx)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to list clusters")
	}

	result := make([]ClusterResponse, len(clusters))
	for i := range clusters {
		result[i] = *toClusterResponse(&clusters[i])
	}
	return result, nil
}

// Get returns a single cluster by ID.
func (s *ClusterService) Get(ctx context.Context, id uint) (*ClusterResponse, error) {
	c, err := s.clusterRepo.GetByID(ctx, id)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, "cluster not found")
	}
	return toClusterResponse(c), nil
}

// Remove removes a cluster and cleans up K8s connections and informers.
func (s *ClusterService) Remove(ctx context.Context, id uint) error {
	c, err := s.clusterRepo.GetByID(ctx, id)
	if err != nil {
		return bizerr.New(bizerr.CodeNotFound, "cluster not found")
	}

	clusterID := c.Name

	// Stop informers.
	s.informerMgr.StopForCluster(clusterID)

	// Remove K8s connection.
	s.clusterManager.Remove(clusterID)

	// Delete from database.
	if err := s.clusterRepo.Delete(ctx, id); err != nil {
		return bizerr.New(bizerr.CodeInternal, "failed to delete cluster")
	}

	s.logger.Info("cluster removed", zap.String("name", c.Name), zap.Uint("id", id))
	return nil
}

// InitClusters loads persisted clusters from the database and reconnects them.
// Called during application startup.
func (s *ClusterService) InitClusters(ctx context.Context) {
	clusters, err := s.clusterRepo.List(ctx)
	if err != nil {
		s.logger.Error("failed to load clusters from database", zap.Error(err))
		return
	}

	for i := range clusters {
		c := &clusters[i]
		clusterID := c.Name

		switch c.AuthType {
		case "kubeconfig":
			if c.KubeconfigEnc == "" {
				s.logger.Warn("cluster has no kubeconfig, skipping", zap.String("name", c.Name))
				continue
			}
			kubeconfig, err := auth.Decrypt(c.KubeconfigEnc, s.encryptKey)
			if err != nil {
				s.logger.Error("failed to decrypt kubeconfig", zap.String("name", c.Name), zap.Error(err))
				c.Status = "unhealthy"
				if updateErr := s.clusterRepo.Update(ctx, c); updateErr != nil {
					s.logger.Error("failed to update cluster status", zap.String("name", c.Name), zap.Error(updateErr))
				}
				continue
			}
			if err := s.clusterManager.Add(clusterID, []byte(kubeconfig)); err != nil {
				s.logger.Error("failed to reconnect cluster", zap.String("name", c.Name), zap.Error(err))
				c.Status = "unhealthy"
				if updateErr := s.clusterRepo.Update(ctx, c); updateErr != nil {
					s.logger.Error("failed to update cluster status", zap.String("name", c.Name), zap.Error(updateErr))
				}
				continue
			}
		case "in-cluster":
			if err := s.clusterManager.AddInCluster(clusterID); err != nil {
				s.logger.Error("failed to reconnect cluster", zap.String("name", c.Name), zap.Error(err))
				c.Status = "unhealthy"
				if updateErr := s.clusterRepo.Update(ctx, c); updateErr != nil {
					s.logger.Error("failed to update cluster status", zap.String("name", c.Name), zap.Error(updateErr))
				}
				continue
			}
		default:
			s.logger.Warn("unknown auth type, skipping cluster", zap.String("name", c.Name), zap.String("authType", c.AuthType))
			continue
		}

		// Start informers.
		dynClient, err := s.clusterManager.DynamicClient(clusterID)
		if err == nil {
			s.informerMgr.StartForCluster(clusterID, dynClient, s.registry.CachedGVRs(), 30*time.Minute)
		}

		s.logger.Info("cluster reconnected", zap.String("name", c.Name), zap.Uint("id", c.ID))
	}
}

// ResolveClusterID resolves a cluster ID string (which may be a numeric DB ID) to
// the cluster name used by the K8s cluster manager.
func (s *ClusterService) ResolveClusterID(ctx context.Context, clusterIDStr string) (string, error) {
	// Try numeric ID first.
	if id, err := strconv.ParseUint(clusterIDStr, 10, 64); err == nil {
		c, err := s.clusterRepo.GetByID(ctx, uint(id))
		if err != nil {
			return "", bizerr.New(bizerr.CodeNotFound, "cluster not found")
		}
		return c.Name, nil
	}
	// Fall back to treating it as a cluster name.
	c, err := s.clusterRepo.GetByName(ctx, clusterIDStr)
	if err != nil {
		return "", bizerr.New(bizerr.CodeNotFound, "cluster not found")
	}
	return c.Name, nil
}

func toClusterResponse(c *model.Cluster) *ClusterResponse {
	return &ClusterResponse{
		ID:          c.ID,
		Name:        c.Name,
		DisplayName: c.DisplayName,
		APIServer:   c.APIServer,
		AuthType:    c.AuthType,
		Status:      c.Status,
		Version:     c.Version,
		CreatedAt:   c.CreatedAt,
	}
}
