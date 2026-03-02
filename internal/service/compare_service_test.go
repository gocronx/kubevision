package service

import (
	"context"
	"errors"
	"testing"

	"github.com/kubevision/kubevision/internal/repository"
)

func TestCompareService_Compare(t *testing.T) {
	k8sRepo := newMockK8sRepo()
	k8sRepo.getResult = &repository.Resource{
		Name:      "nginx",
		Namespace: "default",
		Kind:      "Deployment",
		Raw: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": "nginx",
			},
		},
	}

	svc := NewCompareService(k8sRepo)
	ctx := context.Background()

	req := &CompareRequest{
		Source: CompareTarget{Cluster: "prod", Namespace: "default", Resource: "deployments", Name: "nginx"},
		Target: CompareTarget{Cluster: "dev", Namespace: "default", Resource: "deployments", Name: "nginx"},
	}

	result, err := svc.Compare(ctx, req)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if result.SourceYAML == "" {
		t.Error("expected non-empty source YAML")
	}
	if result.TargetYAML == "" {
		t.Error("expected non-empty target YAML")
	}
	if result.SourceRef == "" {
		t.Error("expected non-empty source ref")
	}
}

func TestCompareService_Compare_SourceNotFound(t *testing.T) {
	k8sRepo := newMockK8sRepo()
	k8sRepo.getErr = errors.New("not found")

	svc := NewCompareService(k8sRepo)
	ctx := context.Background()

	req := &CompareRequest{
		Source: CompareTarget{Cluster: "prod", Namespace: "default", Resource: "deployments", Name: "missing"},
		Target: CompareTarget{Cluster: "dev", Namespace: "default", Resource: "deployments", Name: "nginx"},
	}

	_, err := svc.Compare(ctx, req)
	if err == nil {
		t.Fatal("expected error when source not found")
	}
}
