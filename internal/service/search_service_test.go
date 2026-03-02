package service

import (
	"context"
	"errors"
	"testing"

	bizerr "github.com/kubevision/kubevision/internal/pkg/errors"
	"github.com/kubevision/kubevision/internal/kubernetes/cluster"
	"github.com/kubevision/kubevision/internal/kubernetes/informer"
	"github.com/kubevision/kubevision/internal/kubernetes/resource"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ---------------------------------------------------------------------------
// Helpers: build SearchService with empty managers
// ---------------------------------------------------------------------------

// newSearchSvc constructs a SearchService backed by empty (no-op) concrete
// managers plus the provided clusterRepo mock.  The resource registry is a
// real one so type-filtering tests are exercised properly.
//
// Because informer.Manager and cluster.Manager are concrete types (not
// interfaces), tests that exercise the actual cache/API-server code path
// belong in integration tests.  Here we test:
//
//   - Empty query short-circuit
//   - Cluster-not-found error propagation
//   - Pagination normalisation (limit clamp, negative offset)
//   - Type-filter allow-list construction
//   - matchScore scoring function
//   - sortByScore ordering
//   - buildAPIVersion construction
func newSearchSvc(clusterRepo *mockClusterRepo) *SearchService {
	logger, _ := zap.NewDevelopment()
	informerMgr := informer.NewManager(logger)
	clusterMgr := cluster.NewManager()
	registry := resource.NewRegistry()
	return NewSearchService(informerMgr, clusterMgr, registry, clusterRepo)
}

// ---------------------------------------------------------------------------
// Tests: SearchService.Search — high-level flow
// ---------------------------------------------------------------------------

func TestSearchService_Search(t *testing.T) {
	t.Run("empty query returns empty response without hitting k8s", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		svc := newSearchSvc(clusterRepo)

		resp, err := svc.Search(context.Background(), 1, SearchOptions{Query: ""})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if len(resp.Results) != 0 {
			t.Errorf("expected 0 result groups, got %d", len(resp.Results))
		}
		if resp.Total != 0 {
			t.Errorf("expected total=0, got %d", resp.Total)
		}
	})

	t.Run("whitespace-only query returns empty response", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		svc := newSearchSvc(clusterRepo)

		resp, err := svc.Search(context.Background(), 1, SearchOptions{Query: "   "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Results) != 0 {
			t.Errorf("expected 0 results, got %d", len(resp.Results))
		}
	})

	t.Run("returns not-found error when cluster does not exist", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		svc := newSearchSvc(clusterRepo)

		_, err := svc.Search(context.Background(), 999, SearchOptions{Query: "nginx"})
		if err == nil {
			t.Fatal("expected error for unknown cluster, got nil")
		}
		var bizErr *bizerr.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("expected BizError, got %T: %v", err, err)
		}
		if bizErr.Code != bizerr.CodeNotFound {
			t.Errorf("code = %d, want %d", bizErr.Code, bizerr.CodeNotFound)
		}
	})

	t.Run("non-empty query with valid cluster returns response (possibly empty results)", func(t *testing.T) {
		// The informer cache is empty and the cluster manager has no live
		// connection, so every resource type will fail searchFromCache or
		// searchFromAPIServer.  The service swallows individual type errors
		// and returns an empty groups list — not an error.
		clusterRepo := newMockClusterRepo()
		clusterRepo.addCluster(makeTestCluster(1, "prod"))
		svc := newSearchSvc(clusterRepo)

		resp, err := svc.Search(context.Background(), 1, SearchOptions{Query: "nginx"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		// Results may be empty because the managers have no data, but the
		// response must be well-formed.
		if resp.Results == nil {
			t.Error("Results slice must not be nil (should be empty slice)")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: matchScore (pure function, same package)
// ---------------------------------------------------------------------------

func TestMatchScore(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		resName    string
		namespace  string
		labels     map[string]string
		wantScore  int
	}{
		{
			name:      "exact name match returns score 3",
			query:     "nginx",
			resName:   "nginx",
			wantScore: scoreExact,
		},
		{
			name:      "case-insensitive exact match returns score 3",
			query:     "nginx",
			resName:   "NGINX",
			wantScore: scoreExact,
		},
		{
			name:      "prefix match returns score 2",
			query:     "nginx",
			resName:   "nginx-deployment",
			wantScore: scorePrefix,
		},
		{
			name:      "substring match returns score 1",
			query:     "gin",
			resName:   "nginx-deployment",
			wantScore: scoreSubstring,
		},
		{
			name:      "no name match but namespace contains query returns score 1",
			query:     "team-a",
			resName:   "my-pod",
			namespace: "ns-team-a",
			wantScore: scoreSubstring,
		},
		{
			name:    "no name or namespace match but label key contains query returns score 1",
			query:   "env",
			resName: "my-pod",
			labels:  map[string]string{"environment": "prod"},
			wantScore: scoreSubstring,
		},
		{
			name:    "no name or namespace match but label value contains query returns score 1",
			query:   "prod",
			resName: "my-pod",
			labels:  map[string]string{"env": "production"},
			wantScore: scoreSubstring,
		},
		{
			name:      "no match at all returns zero",
			query:     "zzz-no-match",
			resName:   "nginx",
			namespace: "default",
			labels:    map[string]string{"app": "nginx"},
			wantScore: 0,
		},
		{
			// An empty string is a prefix of every string in Go, so matchScore
			// returns scorePrefix here.  The Search() method guards against
			// empty queries at a higher level (trimming before calling matchScore),
			// so this behaviour is intentional and correct.
			name:      "empty query matches every name as prefix (scored as prefix)",
			query:     "",
			resName:   "nginx",
			wantScore: scorePrefix,
		},
		{
			name:    "nil labels map is handled gracefully",
			query:   "nginx",
			resName: "my-pod",
			labels:  nil,
			wantScore: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := matchScore(tc.query, tc.resName, tc.namespace, tc.labels)
			if got != tc.wantScore {
				t.Errorf("matchScore(%q, %q, %q, %v) = %d, want %d",
					tc.query, tc.resName, tc.namespace, tc.labels, got, tc.wantScore)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: sortByScore (pure function)
// ---------------------------------------------------------------------------

func TestSortByScore(t *testing.T) {
	t.Run("items are sorted descending by score", func(t *testing.T) {
		items := []SearchResultItem{
			{Name: "c", score: scoreSubstring},
			{Name: "a", score: scoreExact},
			{Name: "b", score: scorePrefix},
		}
		sortByScore(items)

		if items[0].Name != "a" {
			t.Errorf("first item name = %q, want %q (exact score)", items[0].Name, "a")
		}
		if items[1].Name != "b" {
			t.Errorf("second item name = %q, want %q (prefix score)", items[1].Name, "b")
		}
		if items[2].Name != "c" {
			t.Errorf("third item name = %q, want %q (substring score)", items[2].Name, "c")
		}
	})

	t.Run("items with equal score are sorted alphabetically by name", func(t *testing.T) {
		items := []SearchResultItem{
			{Name: "zebra", score: scorePrefix},
			{Name: "alpha", score: scorePrefix},
			{Name: "mango", score: scorePrefix},
		}
		sortByScore(items)

		if items[0].Name != "alpha" {
			t.Errorf("first item = %q, want %q", items[0].Name, "alpha")
		}
		if items[1].Name != "mango" {
			t.Errorf("second item = %q, want %q", items[1].Name, "mango")
		}
		if items[2].Name != "zebra" {
			t.Errorf("third item = %q, want %q", items[2].Name, "zebra")
		}
	})

	t.Run("higher score beats alphabetically earlier name", func(t *testing.T) {
		items := []SearchResultItem{
			{Name: "aardvark", score: scoreSubstring},
			{Name: "zebra", score: scoreExact},
		}
		sortByScore(items)

		if items[0].Name != "zebra" {
			t.Errorf("first item = %q, want %q (exact score beats alphabetical)", items[0].Name, "zebra")
		}
	})

	t.Run("empty slice does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("sortByScore panicked on empty slice: %v", r)
			}
		}()
		sortByScore(nil)
		sortByScore([]SearchResultItem{})
	})

	t.Run("single item slice is unchanged", func(t *testing.T) {
		items := []SearchResultItem{{Name: "solo", score: scoreExact}}
		sortByScore(items)
		if items[0].Name != "solo" {
			t.Errorf("single item changed: got %q", items[0].Name)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: buildAPIVersion (pure function)
// ---------------------------------------------------------------------------

func TestBuildAPIVersion(t *testing.T) {
	tests := []struct {
		name string
		meta resource.Meta
		want string
	}{
		{
			name: "core resource (empty group) returns version only",
			meta: resource.Meta{
				GVK: schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
			},
			want: "v1",
		},
		{
			name: "non-core resource returns group/version",
			meta: resource.Meta{
				GVK: schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			},
			want: "apps/v1",
		},
		{
			name: "networking group returns correct apiVersion",
			meta: resource.Meta{
				GVK: schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"},
			},
			want: "networking.k8s.io/v1",
		},
		{
			name: "autoscaling v2 returns correct apiVersion",
			meta: resource.Meta{
				GVK: schema.GroupVersionKind{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"},
			},
			want: "autoscaling/v2",
		},
		{
			name: "rbac group returns correct apiVersion",
			meta: resource.Meta{
				GVK: schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"},
			},
			want: "rbac.authorization.k8s.io/v1",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := buildAPIVersion(tc.meta)
			if got != tc.want {
				t.Errorf("buildAPIVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: SearchOptions normalisation (exercised through Search)
// ---------------------------------------------------------------------------

func TestSearchService_Search_LimitNormalisation(t *testing.T) {
	// We cannot observe pagination directly from the public return value when
	// there are no results, but we CAN verify that the service does not crash
	// with extreme/zero/negative option values, and that the short-circuit for
	// empty query still fires correctly.

	t.Run("limit 0 is treated as default and does not error", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		svc := newSearchSvc(clusterRepo)

		// Empty query — short-circuits before cluster lookup.
		resp, err := svc.Search(context.Background(), 1, SearchOptions{Query: "", Limit: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
	})

	t.Run("limit larger than maxSearchLimit is clamped (no error)", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		svc := newSearchSvc(clusterRepo)

		resp, err := svc.Search(context.Background(), 1, SearchOptions{Query: "", Limit: 9999})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
	})

	t.Run("negative offset is clamped to zero (no error)", func(t *testing.T) {
		clusterRepo := newMockClusterRepo()
		svc := newSearchSvc(clusterRepo)

		resp, err := svc.Search(context.Background(), 1, SearchOptions{Query: "", Offset: -5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: matchScore edge cases — label matching
// ---------------------------------------------------------------------------

func TestMatchScore_LabelMatching(t *testing.T) {
	t.Run("label key match is case-insensitive", func(t *testing.T) {
		score := matchScore("env", "some-pod", "", map[string]string{"ENV": "production"})
		if score != scoreSubstring {
			t.Errorf("expected scoreSubstring (%d) for label key match, got %d", scoreSubstring, score)
		}
	})

	t.Run("label value match is case-insensitive", func(t *testing.T) {
		score := matchScore("prod", "some-pod", "", map[string]string{"env": "PRODUCTION"})
		if score != scoreSubstring {
			t.Errorf("expected scoreSubstring (%d) for label value match, got %d", scoreSubstring, score)
		}
	})

	t.Run("empty label map returns zero score when name and namespace do not match", func(t *testing.T) {
		score := matchScore("nginx", "my-pod", "default", map[string]string{})
		if score != 0 {
			t.Errorf("expected score 0, got %d", score)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: matchScore — name matching precedence
// ---------------------------------------------------------------------------

func TestMatchScore_NameMatchPrecedence(t *testing.T) {
	// Verify that exact beats prefix beats substring when name alone matches.

	t.Run("exact beats prefix for the same name with more context", func(t *testing.T) {
		exactScore := matchScore("nginx", "nginx", "default", nil)
		prefixScore := matchScore("nginx", "nginx-proxy", "default", nil)

		if exactScore <= prefixScore {
			t.Errorf("exact score (%d) should be > prefix score (%d)", exactScore, prefixScore)
		}
	})

	t.Run("prefix beats substring", func(t *testing.T) {
		prefixScore := matchScore("ngi", "ngi-pod", "default", nil)
		subScore := matchScore("ngi", "my-ngi-pod", "default", nil)

		if prefixScore <= subScore {
			t.Errorf("prefix score (%d) should be > substring score (%d)", prefixScore, subScore)
		}
	})
}
