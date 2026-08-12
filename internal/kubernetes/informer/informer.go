package informer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicinformer "k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

// staleDuration is the threshold after which cached data is considered stale.
const staleDuration = 5 * time.Minute

// EventListener is the interface for receiving resource change events.
// This decouples the informer package from the WebSocket hub.
type EventListener interface {
	OnResourceEvent(event ResourceEvent)
}

// ResourceEvent represents a single Kubernetes resource change event.
type ResourceEvent struct {
	Type      string                 `json:"type"` // ADDED | MODIFIED | DELETED
	ClusterID string                 `json:"clusterId"`
	Resource  string                 `json:"resource"`
	Namespace string                 `json:"namespace"`
	Name      string                 `json:"name"`
	Object    map[string]interface{} `json:"object,omitempty"`
}

// clusterRuntime holds the informer factory and lifecycle state for a cluster.
type clusterRuntime struct {
	factory  dynamicinformer.DynamicSharedInformerFactory
	cancel   context.CancelFunc
	synced   bool
	lastSync time.Time
}

type cacheSyncResult struct {
	canceled  bool
	allSynced bool
	failed    []schema.GroupVersionResource
}

func summarizeCacheSync(ctx context.Context, syncs map[schema.GroupVersionResource]bool) cacheSyncResult {
	if ctx.Err() != nil {
		return cacheSyncResult{canceled: true}
	}

	result := cacheSyncResult{allSynced: true}
	for gvr, synced := range syncs {
		if !synced {
			result.allSynced = false
			result.failed = append(result.failed, gvr)
		}
	}
	return result
}

// Manager sets up and manages Kubernetes shared informers for real-time
// resource watching across multiple clusters.
type Manager struct {
	mu        sync.RWMutex
	clusters  map[string]*clusterRuntime
	listeners []EventListener
	logger    *zap.Logger
}

// NewManager creates a new informer Manager.
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		clusters:  make(map[string]*clusterRuntime),
		listeners: make([]EventListener, 0),
		logger:    logger,
	}
}

// AddListener registers an event listener that will be notified of all
// resource change events.
func (m *Manager) AddListener(l EventListener) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, l)
}

// StartForCluster creates a DynamicSharedInformerFactory for the specified
// cluster, registers event handlers for each GVR, starts the factory, and
// waits for cache sync asynchronously.
func (m *Manager) StartForCluster(
	clusterID string,
	client dynamic.Interface,
	resources []schema.GroupVersionResource,
	resyncPeriod time.Duration,
) {
	ctx, cancel := context.WithCancel(context.Background())

	factory := dynamicinformer.NewDynamicSharedInformerFactory(client, resyncPeriod)

	// Register event handlers for each resource.
	for _, gvr := range resources {
		informer := factory.ForResource(gvr).Informer()
		resourceName := gvr.Resource

		informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				m.notify(clusterID, "ADDED", resourceName, obj)
			},
			UpdateFunc: func(_, newObj interface{}) {
				m.notify(clusterID, "MODIFIED", resourceName, newObj)
			},
			DeleteFunc: func(obj interface{}) {
				m.notify(clusterID, "DELETED", resourceName, obj)
			},
		})
	}

	// Store runtime state under lock before starting.
	cr := &clusterRuntime{
		factory:  factory,
		cancel:   cancel,
		synced:   false,
		lastSync: time.Time{},
	}

	m.mu.Lock()
	// Stop existing informer for this cluster if present.
	if existing, ok := m.clusters[clusterID]; ok {
		existing.cancel()
	}
	m.clusters[clusterID] = cr
	m.mu.Unlock()

	// Start the factory (non-blocking).
	factory.Start(ctx.Done())

	// Wait for cache sync asynchronously to avoid holding any locks.
	go func() {
		m.logger.Info("waiting for informer cache sync",
			zap.String("cluster", clusterID),
			zap.Int("resources", len(resources)),
		)

		syncs := factory.WaitForCacheSync(ctx.Done())
		result := summarizeCacheSync(ctx, syncs)
		if result.canceled {
			m.logger.Info("informer cache sync stopped",
				zap.String("cluster", clusterID),
			)
			return
		}

		for _, gvr := range result.failed {
			m.logger.Warn("informer cache sync failed",
				zap.String("cluster", clusterID),
				zap.String("resource", gvr.Resource),
			)
		}

		m.mu.Lock()
		if rt, ok := m.clusters[clusterID]; ok {
			rt.synced = result.allSynced
			rt.lastSync = time.Now()
		}
		m.mu.Unlock()

		if result.allSynced {
			m.logger.Info("informer cache synced",
				zap.String("cluster", clusterID),
			)
		} else {
			m.logger.Warn("informer cache partially synced",
				zap.String("cluster", clusterID),
			)
		}
	}()
}

// StopForCluster stops all informers for the specified cluster by cancelling
// its context.
func (m *Manager) StopForCluster(clusterID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cr, ok := m.clusters[clusterID]; ok {
		cr.cancel()
		delete(m.clusters, clusterID)
		m.logger.Info("informer stopped", zap.String("cluster", clusterID))
	}
}

// List retrieves resources from the informer cache. It returns a stale flag
// indicating whether the cache data may be outdated (no sync in the last
// 5 minutes or cache never synced).
func (m *Manager) List(clusterID string, gvr schema.GroupVersionResource, namespace string) (
	[]unstructured.Unstructured, bool /* stale */, error,
) {
	m.mu.RLock()
	cr, ok := m.clusters[clusterID]
	if !ok {
		m.mu.RUnlock()
		return nil, false, fmt.Errorf("cluster %s not found", clusterID)
	}
	stale := !cr.synced || time.Since(cr.lastSync) > staleDuration
	m.mu.RUnlock()

	lister := cr.factory.ForResource(gvr).Lister()

	var objs []runtime.Object
	var err error
	if namespace != "" {
		objs, err = lister.ByNamespace(namespace).List(labels.Everything())
	} else {
		objs, err = lister.List(labels.Everything())
	}
	if err != nil {
		return nil, stale, fmt.Errorf("list %s in cluster %s: %w", gvr.Resource, clusterID, err)
	}

	result := make([]unstructured.Unstructured, 0, len(objs))
	for _, obj := range objs {
		u, ok := obj.(*unstructured.Unstructured)
		if !ok {
			continue
		}
		result = append(result, *u)
	}

	return result, stale, nil
}

// Get retrieves a single resource from the informer cache by namespace and name.
func (m *Manager) Get(clusterID string, gvr schema.GroupVersionResource, namespace, name string) (
	*unstructured.Unstructured, error,
) {
	m.mu.RLock()
	cr, ok := m.clusters[clusterID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("cluster %s not found", clusterID)
	}

	lister := cr.factory.ForResource(gvr).Lister()

	var obj runtime.Object
	var err error
	if namespace != "" {
		obj, err = lister.ByNamespace(namespace).Get(name)
	} else {
		obj, err = lister.Get(name)
	}
	if err != nil {
		return nil, fmt.Errorf("get %s/%s in cluster %s: %w", gvr.Resource, name, clusterID, err)
	}

	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("unexpected object type for %s/%s", gvr.Resource, name)
	}

	return u, nil
}

// notify dispatches a resource event to all registered listeners. Notification
// is non-blocking: each listener is invoked in its own goroutine to prevent
// slow listeners from stalling the informer event pipeline.
func (m *Manager) notify(clusterID, eventType, resource string, obj interface{}) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		// Handle DeletedFinalStateUnknown from the informer cache.
		if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
			u, ok = tombstone.Obj.(*unstructured.Unstructured)
			if !ok {
				return
			}
		} else {
			return
		}
	}

	event := ResourceEvent{
		Type:      eventType,
		ClusterID: clusterID,
		Resource:  resource,
		Namespace: u.GetNamespace(),
		Name:      u.GetName(),
		Object:    u.Object,
	}

	m.mu.RLock()
	listeners := make([]EventListener, len(m.listeners))
	copy(listeners, m.listeners)
	m.mu.RUnlock()

	for _, l := range listeners {
		l := l
		go l.OnResourceEvent(event)
	}
}

// StopAll stops all cluster informers. This is used for graceful shutdown.
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, cr := range m.clusters {
		cr.cancel()
		m.logger.Info("informer stopped", zap.String("cluster", id))
	}
	m.clusters = make(map[string]*clusterRuntime)
}
