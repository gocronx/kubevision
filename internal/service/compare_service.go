package service

import (
	"context"
	"fmt"

	"sigs.k8s.io/yaml"

	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/repository"
)

// CompareTarget identifies a single Kubernetes resource to fetch.
type CompareTarget struct {
	Cluster   string `json:"cluster"   binding:"required"`
	Namespace string `json:"namespace"`
	Resource  string `json:"resource"  binding:"required"`
	Name      string `json:"name"      binding:"required"`
}

// CompareRequest is the API request body for a cross-cluster comparison.
type CompareRequest struct {
	Source CompareTarget `json:"source" binding:"required"`
	Target CompareTarget `json:"target" binding:"required"`
}

// CompareResult holds the YAML representations of the two compared resources.
type CompareResult struct {
	SourceYAML string `json:"sourceYaml"`
	TargetYAML string `json:"targetYaml"`
	SourceRef  string `json:"sourceRef"`
	TargetRef  string `json:"targetRef"`
}

// CompareService fetches two Kubernetes resources and returns their YAML.
type CompareService struct {
	k8sRepo repository.K8sResourceRepo
}

// NewCompareService creates a new CompareService.
func NewCompareService(k8sRepo repository.K8sResourceRepo) *CompareService {
	return &CompareService{k8sRepo: k8sRepo}
}

// Compare fetches the source and target resources and converts them to YAML
// for side-by-side diffing in the frontend.
func (s *CompareService) Compare(ctx context.Context, req *CompareRequest) (*CompareResult, error) {
	sourceRes, err := s.k8sRepo.Get(ctx, req.Source.Cluster, req.Source.Resource, req.Source.Namespace, req.Source.Name)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("source resource not found: %v", err))
	}

	targetRes, err := s.k8sRepo.Get(ctx, req.Target.Cluster, req.Target.Resource, req.Target.Namespace, req.Target.Name)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeNotFound, fmt.Sprintf("target resource not found: %v", err))
	}

	sourceYAML, err := rawToYAML(sourceRes.Raw)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to convert source resource to YAML")
	}

	targetYAML, err := rawToYAML(targetRes.Raw)
	if err != nil {
		return nil, bizerr.New(bizerr.CodeInternal, "failed to convert target resource to YAML")
	}

	sourceRef := fmt.Sprintf("%s/%s/%s/%s", req.Source.Cluster, req.Source.Namespace, req.Source.Resource, req.Source.Name)
	targetRef := fmt.Sprintf("%s/%s/%s/%s", req.Target.Cluster, req.Target.Namespace, req.Target.Resource, req.Target.Name)

	return &CompareResult{
		SourceYAML: sourceYAML,
		TargetYAML: targetYAML,
		SourceRef:  sourceRef,
		TargetRef:  targetRef,
	}, nil
}

// rawToYAML converts a raw map[string]interface{} (from Resource.Raw) to a
// YAML string using the sigs.k8s.io/yaml library which respects JSON tags.
func rawToYAML(raw map[string]interface{}) (string, error) {
	if raw == nil {
		return "", nil
	}
	b, err := yaml.Marshal(raw)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
