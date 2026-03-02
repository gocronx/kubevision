package service

// mocks_test.go — shared mock implementations used across multiple service
// test files in this package.  All types are unexported and test-only.

import (
	"context"
	"errors"
	"time"

	"github.com/kubevision/kubevision/internal/model"
	"github.com/kubevision/kubevision/internal/repository"
)

// ---------------------------------------------------------------------------
// mockClusterRepo implements repository.ClusterRepo
// ---------------------------------------------------------------------------

type mockClusterRepo struct {
	clusters   map[uint]*model.Cluster
	byName     map[string]*model.Cluster
	nextID     uint
	createErr  error
	updateErr  error
	deleteErr  error
	listErr    error
}

func newMockClusterRepo() *mockClusterRepo {
	return &mockClusterRepo{
		clusters: make(map[uint]*model.Cluster),
		byName:   make(map[string]*model.Cluster),
		nextID:   1,
	}
}

func (m *mockClusterRepo) addCluster(c *model.Cluster) {
	if c.ID == 0 {
		c.ID = m.nextID
		m.nextID++
	}
	cp := *c
	m.clusters[c.ID] = &cp
	m.byName[c.Name] = &cp
}

func (m *mockClusterRepo) Create(_ context.Context, c *model.Cluster) error {
	if m.createErr != nil {
		return m.createErr
	}
	c.ID = m.nextID
	m.nextID++
	cp := *c
	m.clusters[c.ID] = &cp
	m.byName[c.Name] = &cp
	return nil
}

func (m *mockClusterRepo) GetByID(_ context.Context, id uint) (*model.Cluster, error) {
	c, ok := m.clusters[id]
	if !ok {
		return nil, errors.New("cluster not found")
	}
	cp := *c
	return &cp, nil
}

func (m *mockClusterRepo) GetByName(_ context.Context, name string) (*model.Cluster, error) {
	c, ok := m.byName[name]
	if !ok {
		return nil, errors.New("cluster not found")
	}
	cp := *c
	return &cp, nil
}

func (m *mockClusterRepo) Update(_ context.Context, c *model.Cluster) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	cp := *c
	m.clusters[c.ID] = &cp
	m.byName[c.Name] = &cp
	return nil
}

func (m *mockClusterRepo) Delete(_ context.Context, id uint) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if c, ok := m.clusters[id]; ok {
		delete(m.byName, c.Name)
	}
	delete(m.clusters, id)
	return nil
}

func (m *mockClusterRepo) List(_ context.Context) ([]model.Cluster, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	result := make([]model.Cluster, 0, len(m.clusters))
	for _, c := range m.clusters {
		result = append(result, *c)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// mockK8sRepo implements repository.K8sResourceRepo
// ---------------------------------------------------------------------------

type mockK8sRepo struct {
	// For List
	listResult        *repository.ResourceList
	listErr           error
	lastListClusterID string
	lastListKind      string
	lastListNamespace string

	// For Get
	getResult *repository.Resource
	getErr    error

	// For Create
	createResult *repository.Resource
	createErr    error

	// For Update
	updateResult *repository.Resource
	updateErr    error

	// For Delete
	deleteErr error

	// For Patch
	patchResult *repository.Resource
	patchErr    error

	// For DryRunCreate
	dryRunCreateResult *repository.Resource
	dryRunCreateErr    error

	// For DryRunUpdate
	dryRunUpdateCurrent  *repository.Resource
	dryRunUpdateProposed *repository.Resource
	dryRunUpdateErr      error
}

func newMockK8sRepo() *mockK8sRepo {
	return &mockK8sRepo{}
}

func (m *mockK8sRepo) List(_ context.Context, clusterID, kind, namespace string, _ repository.ListOptions) (*repository.ResourceList, error) {
	m.lastListClusterID = clusterID
	m.lastListKind = kind
	m.lastListNamespace = namespace
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.listResult != nil {
		return m.listResult, nil
	}
	return &repository.ResourceList{Items: []repository.Resource{}}, nil
}

func (m *mockK8sRepo) Get(_ context.Context, _, _, _, _ string) (*repository.Resource, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getResult, nil
}

func (m *mockK8sRepo) Create(_ context.Context, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createResult, nil
}

func (m *mockK8sRepo) Update(_ context.Context, _, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return m.updateResult, nil
}

func (m *mockK8sRepo) Delete(_ context.Context, _, _, _, _ string) error {
	return m.deleteErr
}

func (m *mockK8sRepo) Patch(_ context.Context, _, _, _, _ string, _ []byte) (*repository.Resource, error) {
	if m.patchErr != nil {
		return nil, m.patchErr
	}
	return m.patchResult, nil
}

func (m *mockK8sRepo) DryRunCreate(_ context.Context, _, _, _ string, _ map[string]interface{}) (*repository.Resource, error) {
	if m.dryRunCreateErr != nil {
		return nil, m.dryRunCreateErr
	}
	return m.dryRunCreateResult, nil
}

func (m *mockK8sRepo) DryRunUpdate(_ context.Context, _, _, _, _ string, _ map[string]interface{}) (*repository.Resource, *repository.Resource, error) {
	if m.dryRunUpdateErr != nil {
		return nil, nil, m.dryRunUpdateErr
	}
	return m.dryRunUpdateCurrent, m.dryRunUpdateProposed, nil
}

// ---------------------------------------------------------------------------
// Helper: build a minimal test cluster model
// ---------------------------------------------------------------------------

func makeTestCluster(id uint, name string) *model.Cluster {
	return &model.Cluster{
		ID:        id,
		Name:      name,
		APIServer: "https://" + name + ".example.com:6443",
		AuthType:  "kubeconfig",
		Status:    "healthy",
		CreatedAt: time.Now(),
	}
}
