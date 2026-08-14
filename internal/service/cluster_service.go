package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/auth"
	"github.com/gocronx/kubevision/internal/kubernetes/cluster"
	"github.com/gocronx/kubevision/internal/kubernetes/informer"
	"github.com/gocronx/kubevision/internal/kubernetes/resource"
	"github.com/gocronx/kubevision/internal/model"
	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"github.com/gocronx/kubevision/internal/repository"
)

// ClusterResponse is the API response for a cluster.
type ClusterResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	APIServer string    `json:"apiServer"`
	AuthType  string    `json:"authType"`
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
}

// AddClusterRequest is the request body for adding a cluster.
type AddClusterRequest struct {
	Name       string `json:"name" binding:"required"`
	AuthType   string `json:"authType" binding:"required"` // kubeconfig | in-cluster
	Kubeconfig string `json:"kubeconfig"`
}

// ClusterService encapsulates business logic for Kubernetes cluster management.
type ClusterService struct {
	mu             sync.Mutex
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
	s.mu.Lock()
	defer s.mu.Unlock()
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, bizerr.New(bizerr.CodeParamInvalid, "cluster name is required")
	}

	// Check name uniqueness.
	if existing, _ := s.clusterRepo.GetByName(ctx, req.Name); existing != nil {
		return nil, bizerr.New(bizerr.CodeConflict, fmt.Sprintf("cluster %q already exists", req.Name))
	}

	// Register the cluster with the K8s cluster manager.
	clusterID := req.Name // use name as cluster ID for simplicity
	switch req.AuthType {
	case "kubeconfig":
		if strings.TrimSpace(req.Kubeconfig) == "" {
			return nil, bizerr.New(bizerr.CodeParamInvalid, "kubeconfig is required for auth type 'kubeconfig'")
		}
		if err := s.clusterManager.Add(clusterID, []byte(req.Kubeconfig)); err != nil {
			return nil, bizerr.New(bizerr.CodeK8sUnavailable, fmt.Sprintf("failed to connect: %s", err.Error()))
		}
	case "in-cluster":
		if os.Getenv("KUBERNETES_SERVICE_HOST") == "" || os.Getenv("KUBERNETES_SERVICE_PORT") == "" {
			return nil, bizerr.New(bizerr.CodeParamInvalid, "in-cluster authentication is only available when KubeVision runs inside Kubernetes; use a kubeconfig for local development")
		}
		if err := s.clusterManager.AddInCluster(clusterID); err != nil {
			return nil, bizerr.New(bizerr.CodeK8sUnavailable, fmt.Sprintf("failed to connect: %s", err.Error()))
		}
	default:
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("unsupported auth type: %s", req.AuthType))
	}

	clusterInfo, err := s.clusterManager.Probe(ctx, clusterID)
	if err != nil {
		s.clusterManager.Remove(clusterID)
		return nil, bizerr.New(bizerr.CodeK8sUnavailable, fmt.Sprintf("cluster health check failed: %s", err.Error()))
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
		APIServer:     clusterInfo.APIServer,
		AuthType:      req.AuthType,
		KubeconfigEnc: kubeconfigEnc,
		Status:        "healthy",
		Version:       clusterInfo.Version,
	}
	if err := s.clusterRepo.Create(ctx, m); err != nil {
		// Rollback K8s connection on DB failure.
		s.clusterManager.Remove(clusterID)
		s.logger.Error("failed to save cluster", zap.String("name", req.Name), zap.Error(err))
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
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

	clusters, err := s.clusterRepo.List(ctx)
	if err != nil {
		s.logger.Error("failed to load clusters from database", zap.Error(err))
		return
	}

	for i := range clusters {
		c := &clusters[i]
		if err := s.connectPersistedCluster(c); err != nil {
			s.logger.Error("failed to reconnect cluster", zap.String("name", c.Name), zap.Error(err))
			c.Status = "unhealthy"
			if updateErr := s.clusterRepo.Update(ctx, c); updateErr != nil {
				s.logger.Error("failed to update cluster status", zap.String("name", c.Name), zap.Error(updateErr))
			}
			continue
		}

		if err := s.refreshClusterHealth(ctx, c, true); err != nil {
			s.logger.Warn("cluster health check failed", zap.String("name", c.Name), zap.Error(err))
			continue
		}

		s.logger.Info("cluster reconnected", zap.String("name", c.Name), zap.Uint("id", c.ID))
	}
}

func (s *ClusterService) connectPersistedCluster(c *model.Cluster) error {
	switch c.AuthType {
	case "kubeconfig":
		if c.KubeconfigEnc == "" {
			return fmt.Errorf("cluster has no kubeconfig")
		}
		kubeconfig, err := auth.Decrypt(c.KubeconfigEnc, s.encryptKey)
		if err != nil {
			return fmt.Errorf("decrypt kubeconfig: %w", err)
		}
		if err := s.clusterManager.Add(c.Name, []byte(kubeconfig)); err != nil {
			return err
		}
	case "in-cluster":
		if err := s.clusterManager.AddInCluster(c.Name); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported auth type %q", c.AuthType)
	}
	return nil
}

// StartHealthChecks periodically refreshes persisted cluster health until the
// context is canceled. InitClusters performs the initial check at startup.
func (s *ClusterService) StartHealthChecks(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.RefreshHealth(ctx)
		}
	}
}

// RefreshHealth probes every registered cluster and persists status changes.
func (s *ClusterService) RefreshHealth(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusters, err := s.clusterRepo.List(ctx)
	if err != nil {
		s.logger.Error("failed to load clusters for health check", zap.Error(err))
		return
	}

	for i := range clusters {
		if err := s.refreshClusterHealth(ctx, &clusters[i], false); err != nil {
			s.logger.Warn("cluster health check failed", zap.String("name", clusters[i].Name), zap.Error(err))
		}
	}
}

func (s *ClusterService) refreshClusterHealth(ctx context.Context, c *model.Cluster, startInformer bool) error {
	if _, err := s.clusterManager.RESTConfig(c.Name); err != nil {
		if err := s.connectPersistedCluster(c); err != nil {
			return fmt.Errorf("restore cluster connection: %w", err)
		}
		startInformer = true
		s.logger.Info("cluster connection restored", zap.String("name", c.Name))
	}

	info, probeErr := s.clusterManager.Probe(ctx, c.Name)
	if probeErr != nil {
		s.informerMgr.StopForCluster(c.Name)
		if c.Status != "unhealthy" {
			c.Status = "unhealthy"
			if err := s.clusterRepo.Update(ctx, c); err != nil {
				return fmt.Errorf("persist unhealthy cluster status: %w", err)
			}
		}
		return probeErr
	}

	wasHealthy := c.Status == "healthy"
	changed := !wasHealthy || c.APIServer != info.APIServer || c.Version != info.Version
	c.Status = "healthy"
	c.APIServer = info.APIServer
	c.Version = info.Version
	if changed {
		if err := s.clusterRepo.Update(ctx, c); err != nil {
			return fmt.Errorf("persist healthy cluster status: %w", err)
		}
	}

	if startInformer || !wasHealthy {
		dynClient, err := s.clusterManager.DynamicClient(c.Name)
		if err != nil {
			return err
		}
		s.informerMgr.StartForCluster(c.Name, dynClient, s.registry.CachedGVRs(), 30*time.Minute)
	}
	return nil
}

// ResolveClusterID resolves a cluster ID string (which may be a numeric DB ID) to
// the cluster name used by the K8s cluster manager.
func (s *ClusterService) ResolveClusterID(ctx context.Context, clusterIDStr string) (string, error) {
	// Try numeric ID first.
	if id, err := strconv.ParseUint(clusterIDStr, 10, strconv.IntSize); err == nil {
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
		ID:        c.ID,
		Name:      c.Name,
		APIServer: c.APIServer,
		AuthType:  c.AuthType,
		Status:    c.Status,
		Version:   c.Version,
		CreatedAt: c.CreatedAt,
	}
}
