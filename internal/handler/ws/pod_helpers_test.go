package ws

import (
	"context"
	"errors"
	"testing"

	"github.com/kubevision/kubevision/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Stub ClusterRepo
// ---------------------------------------------------------------------------

// stubClusterRepo is a minimal in-memory implementation of repository.ClusterRepo
// used solely for unit-testing the ws package without a real database.
type stubClusterRepo struct {
	byID   map[uint]*model.Cluster
	byName map[string]*model.Cluster
}

func newStubClusterRepo() *stubClusterRepo {
	return &stubClusterRepo{
		byID:   make(map[uint]*model.Cluster),
		byName: make(map[string]*model.Cluster),
	}
}

func (s *stubClusterRepo) add(c *model.Cluster) {
	s.byID[c.ID] = c
	s.byName[c.Name] = c
}

func (s *stubClusterRepo) Create(_ context.Context, _ *model.Cluster) error {
	return errors.New("stub: Create not implemented")
}

func (s *stubClusterRepo) GetByID(_ context.Context, id uint) (*model.Cluster, error) {
	if c, ok := s.byID[id]; ok {
		return c, nil
	}
	return nil, errors.New("cluster not found")
}

func (s *stubClusterRepo) GetByName(_ context.Context, name string) (*model.Cluster, error) {
	if c, ok := s.byName[name]; ok {
		return c, nil
	}
	return nil, errors.New("cluster not found")
}

func (s *stubClusterRepo) Update(_ context.Context, _ *model.Cluster) error {
	return errors.New("stub: Update not implemented")
}

func (s *stubClusterRepo) Delete(_ context.Context, _ uint) error {
	return errors.New("stub: Delete not implemented")
}

func (s *stubClusterRepo) List(_ context.Context) ([]model.Cluster, error) {
	return nil, errors.New("stub: List not implemented")
}

// ---------------------------------------------------------------------------
// Tests for resolveClusterName
// ---------------------------------------------------------------------------

func TestResolveClusterName_NumericIDLooksUpByID(t *testing.T) {
	repo := newStubClusterRepo()
	repo.add(&model.Cluster{ID: 42, Name: "prod-cluster"})

	name, err := resolveClusterName(context.Background(), "42", repo)
	require.NoError(t, err)
	assert.Equal(t, "prod-cluster", name)
}

func TestResolveClusterName_StringIDFallsBackToGetByName(t *testing.T) {
	repo := newStubClusterRepo()
	repo.add(&model.Cluster{ID: 1, Name: "dev-cluster"})

	name, err := resolveClusterName(context.Background(), "dev-cluster", repo)
	require.NoError(t, err)
	assert.Equal(t, "dev-cluster", name)
}

func TestResolveClusterName_NumericIDNotFound_ReturnsError(t *testing.T) {
	repo := newStubClusterRepo()
	// No cluster with ID 99 registered.

	_, err := resolveClusterName(context.Background(), "99", repo)
	require.Error(t, err)
}

func TestResolveClusterName_StringNameNotFound_ReturnsError(t *testing.T) {
	repo := newStubClusterRepo()
	// No cluster with the given name.

	_, err := resolveClusterName(context.Background(), "nonexistent", repo)
	require.Error(t, err)
}

func TestResolveClusterName_ZeroNumericID(t *testing.T) {
	repo := newStubClusterRepo()
	// "0" is a valid uint; test that we attempt a DB lookup and propagate not-found.

	_, err := resolveClusterName(context.Background(), "0", repo)
	require.Error(t, err, "ID 0 should not match any cluster")
}

func TestResolveClusterName_LargeNumericID(t *testing.T) {
	repo := newStubClusterRepo()
	repo.add(&model.Cluster{ID: 100000, Name: "huge-id-cluster"})

	name, err := resolveClusterName(context.Background(), "100000", repo)
	require.NoError(t, err)
	assert.Equal(t, "huge-id-cluster", name)
}

func TestResolveClusterName_NegativeNumericString_FallsBackToGetByName(t *testing.T) {
	repo := newStubClusterRepo()
	// "-1" cannot be parsed as uint64, so the code falls back to GetByName.

	_, err := resolveClusterName(context.Background(), "-1", repo)
	require.Error(t, err, "negative string should fall back to name lookup and fail")
}

func TestResolveClusterName_EmptyString_FallsBackToGetByName(t *testing.T) {
	repo := newStubClusterRepo()

	_, err := resolveClusterName(context.Background(), "", repo)
	require.Error(t, err, "empty string should fall back to name lookup and fail")
}

func TestResolveClusterName_MultipleClustersByID(t *testing.T) {
	repo := newStubClusterRepo()
	repo.add(&model.Cluster{ID: 1, Name: "cluster-one"})
	repo.add(&model.Cluster{ID: 2, Name: "cluster-two"})
	repo.add(&model.Cluster{ID: 3, Name: "cluster-three"})

	for id, expected := range map[string]string{
		"1": "cluster-one",
		"2": "cluster-two",
		"3": "cluster-three",
	} {
		name, err := resolveClusterName(context.Background(), id, repo)
		require.NoError(t, err)
		assert.Equal(t, expected, name, "cluster ID %s should resolve to %s", id, expected)
	}
}

// ---------------------------------------------------------------------------
// Tests for sentinel errors
// ---------------------------------------------------------------------------

func TestSentinelErrors_NotNilAndDistinct(t *testing.T) {
	assert.NotNil(t, errAccountDisabled)
	assert.NotNil(t, errTokenRevoked)
	assert.NotEqual(t, errAccountDisabled, errTokenRevoked,
		"sentinel errors must be distinct so callers can distinguish them")
}

func TestSentinelErrors_ErrorMessages(t *testing.T) {
	assert.Equal(t, "account is disabled", errAccountDisabled.Error())
	assert.Equal(t, "token has been revoked", errTokenRevoked.Error())
}
