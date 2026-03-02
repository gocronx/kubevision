package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kubevision/kubevision/internal/kubernetes/resource"
	"github.com/kubevision/kubevision/internal/model"
	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/repository"
)

// ResourceService encapsulates business logic for Kubernetes resource operations.
// It coordinates between the K8sResourceRepo and applies business rules such as
// cluster validation, resource type resolution, and error wrapping.
type ResourceService struct {
	k8sRepo     repository.K8sResourceRepo
	registry    *resource.Registry
	clusterRepo repository.ClusterRepo
}

// NewResourceService creates a new ResourceService with the given dependencies.
func NewResourceService(
	k8sRepo repository.K8sResourceRepo,
	registry *resource.Registry,
	clusterRepo repository.ClusterRepo,
) *ResourceService {
	return &ResourceService{
		k8sRepo:     k8sRepo,
		registry:    registry,
		clusterRepo: clusterRepo,
	}
}

// ListResources lists resources of the given kind in the specified cluster and namespace.
func (s *ResourceService) ListResources(
	ctx context.Context,
	clusterID uint,
	resourceName string,
	namespace string,
	opts repository.ListOptions,
) (*repository.ResourceList, error) {
	// Validate the cluster exists.
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}

	// Validate the resource type is registered.
	if _, ok := s.registry.Get(resourceName); !ok {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("unknown resource type: %s", resourceName))
	}

	result, err := s.k8sRepo.List(ctx, clusterKey(cluster), resourceName, namespace, opts)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, fmt.Sprintf("failed to list %s: %s", resourceName, err.Error()))
	}

	return result, nil
}

// GetResource retrieves a single resource by kind, namespace, and name.
func (s *ResourceService) GetResource(
	ctx context.Context,
	clusterID uint,
	resourceName string,
	namespace string,
	name string,
) (*repository.Resource, error) {
	// Validate the cluster exists.
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}

	// Validate the resource type is registered.
	if _, ok := s.registry.Get(resourceName); !ok {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("unknown resource type: %s", resourceName))
	}

	res, err := s.k8sRepo.Get(ctx, clusterKey(cluster), resourceName, namespace, name)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("failed to get %s/%s: %s", resourceName, name, err.Error()))
	}

	return res, nil
}

// CreateResource creates a new resource from a JSON body.
func (s *ResourceService) CreateResource(
	ctx context.Context,
	clusterID uint,
	resourceName string,
	namespace string,
	body []byte,
) (*repository.Resource, error) {
	// Validate the cluster exists.
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}

	// Validate the resource type is registered.
	if _, ok := s.registry.Get(resourceName); !ok {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("unknown resource type: %s", resourceName))
	}

	// Parse the JSON body into an unstructured object.
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("invalid JSON body: %s", err.Error()))
	}

	res, err := s.k8sRepo.Create(ctx, clusterKey(cluster), resourceName, namespace, obj)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, fmt.Sprintf("failed to create %s: %s", resourceName, err.Error()))
	}

	return res, nil
}

// UpdateResource replaces an existing resource with the provided JSON body.
func (s *ResourceService) UpdateResource(
	ctx context.Context,
	clusterID uint,
	resourceName string,
	namespace string,
	name string,
	body []byte,
) (*repository.Resource, error) {
	// Validate the cluster exists.
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}

	// Validate the resource type is registered.
	if _, ok := s.registry.Get(resourceName); !ok {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("unknown resource type: %s", resourceName))
	}

	// Parse the JSON body into an unstructured object.
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("invalid JSON body: %s", err.Error()))
	}

	res, err := s.k8sRepo.Update(ctx, clusterKey(cluster), resourceName, namespace, name, obj)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, fmt.Sprintf("failed to update %s/%s: %s", resourceName, name, err.Error()))
	}

	return res, nil
}

// DeleteResource removes a resource by kind, namespace, and name.
func (s *ResourceService) DeleteResource(
	ctx context.Context,
	clusterID uint,
	resourceName string,
	namespace string,
	name string,
) error {
	// Validate the cluster exists.
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}

	// Validate the resource type is registered.
	if _, ok := s.registry.Get(resourceName); !ok {
		return bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("unknown resource type: %s", resourceName))
	}

	if err := s.k8sRepo.Delete(ctx, clusterKey(cluster), resourceName, namespace, name); err != nil {
		return bizerr.New(bizerr.CodeInternal, fmt.Sprintf("failed to delete %s/%s: %s", resourceName, name, err.Error()))
	}

	return nil
}

// PatchResource applies a strategic merge patch to a resource.
func (s *ResourceService) PatchResource(
	ctx context.Context,
	clusterID uint,
	resourceName string,
	namespace string,
	name string,
	patchData []byte,
) (*repository.Resource, error) {
	// Validate the cluster exists.
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}

	// Validate the resource type is registered.
	if _, ok := s.registry.Get(resourceName); !ok {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("unknown resource type: %s", resourceName))
	}

	// Validate the patch data is valid JSON.
	var check map[string]interface{}
	if err := json.Unmarshal(patchData, &check); err != nil {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("invalid patch JSON: %s", err.Error()))
	}

	res, err := s.k8sRepo.Patch(ctx, clusterKey(cluster), resourceName, namespace, name, patchData)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, fmt.Sprintf("failed to patch %s/%s: %s", resourceName, name, err.Error()))
	}

	return res, nil
}

// DryRunResult holds the result of a dry-run operation.
type DryRunResult struct {
	// Current is the live resource before the change. Nil for create dry-runs.
	Current *repository.Resource `json:"current,omitempty"`
	// Proposed is what the resource would look like after the change.
	Proposed *repository.Resource `json:"proposed"`
	// Valid indicates whether the API server accepted the dry-run without errors.
	Valid bool `json:"valid"`
	// Errors contains validation messages when Valid is false.
	Errors []string `json:"errors,omitempty"`
}

// DryRunCreateResource performs a server-side dry-run create. It returns the
// resource as the API server would have stored it (with defaults filled in)
// without actually creating it.
func (s *ResourceService) DryRunCreateResource(
	ctx context.Context,
	clusterID uint,
	resourceName string,
	namespace string,
	body []byte,
) (*DryRunResult, error) {
	// Validate the cluster exists.
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}

	// Validate the resource type is registered.
	if _, ok := s.registry.Get(resourceName); !ok {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("unknown resource type: %s", resourceName))
	}

	// Parse the JSON body into an unstructured object.
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("invalid JSON body: %s", err.Error()))
	}

	proposed, err := s.k8sRepo.DryRunCreate(ctx, clusterKey(cluster), resourceName, namespace, obj)
	if err != nil {
		// Return a validation-failure result rather than a hard error so the
		// caller can surface the message in the UI.
		return &DryRunResult{
			Valid:  false,
			Errors: []string{err.Error()},
		}, nil
	}

	return &DryRunResult{
		Proposed: proposed,
		Valid:    true,
	}, nil
}

// DryRunUpdateResource performs a server-side dry-run update. It returns both
// the current live resource and what it would look like after the update.
func (s *ResourceService) DryRunUpdateResource(
	ctx context.Context,
	clusterID uint,
	resourceName string,
	namespace string,
	name string,
	body []byte,
) (*DryRunResult, error) {
	// Validate the cluster exists.
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("cluster %d not found", clusterID))
	}

	// Validate the resource type is registered.
	if _, ok := s.registry.Get(resourceName); !ok {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("unknown resource type: %s", resourceName))
	}

	// Parse the JSON body into an unstructured object.
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, bizerr.New(bizerr.CodeParamInvalid, fmt.Sprintf("invalid JSON body: %s", err.Error()))
	}

	current, proposed, err := s.k8sRepo.DryRunUpdate(ctx, clusterKey(cluster), resourceName, namespace, name, obj)
	if err != nil {
		// Return a validation-failure result rather than a hard error.
		return &DryRunResult{
			Valid:  false,
			Errors: []string{err.Error()},
		}, nil
	}

	return &DryRunResult{
		Current:  current,
		Proposed: proposed,
		Valid:    true,
	}, nil
}

// clusterKey converts a model.Cluster to the string identifier used by the
// cluster manager and informer manager. We use the cluster's Name field as the
// key, which matches how ClusterService.Add() registers clusters.
func clusterKey(c *model.Cluster) string {
	return c.Name
}
