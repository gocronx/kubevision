package packages

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/yaml"
)

type RESTConfigProvider interface {
	RESTConfig(string) (*rest.Config, error)
}

type HelmAdapter struct{ clusters RESTConfigProvider }

func NewHelmAdapter(clusters RESTConfigProvider) *HelmAdapter {
	return &HelmAdapter{clusters: clusters}
}

func (h *HelmAdapter) List(_ context.Context, cluster string, opts ListOptions) ([]Release, error) {
	cfg, err := h.configuration(cluster, opts.Namespace)
	if err != nil {
		return nil, err
	}
	client := action.NewList(cfg)
	client.AllNamespaces = opts.Namespace == ""
	client.StateMask = action.ListAll
	client.Limit = opts.Limit
	client.Selector = opts.Label
	items, err := client.Run()
	if err != nil {
		return nil, err
	}
	out := mapReleases(items)
	if opts.State != "" {
		filtered := out[:0]
		for _, item := range out {
			if strings.EqualFold(item.Status, opts.State) {
				filtered = append(filtered, item)
			}
		}
		out = filtered
	}
	return out, nil
}

func (h *HelmAdapter) Get(_ context.Context, cluster, namespace, name string, revision int) (*Release, error) {
	cfg, err := h.configuration(cluster, namespace)
	if err != nil {
		return nil, err
	}
	if revision > 0 {
		client := action.NewGet(cfg)
		client.Version = revision
		item, err := client.Run(name)
		if err != nil {
			return nil, err
		}
		mapped := mapRelease(item)
		return &mapped, nil
	}
	item, err := cfg.Releases.Last(name)
	if err != nil {
		return nil, err
	}
	mapped := mapRelease(item)
	return &mapped, nil
}

func (h *HelmAdapter) History(_ context.Context, cluster, namespace, name string) ([]Release, error) {
	cfg, err := h.configuration(cluster, namespace)
	if err != nil {
		return nil, err
	}
	client := action.NewHistory(cfg)
	client.Max = 256
	items, err := client.Run(name)
	if err != nil {
		return nil, err
	}
	out := mapReleases(items)
	sort.Slice(out, func(i, j int) bool { return out[i].Revision > out[j].Revision })
	return out, nil
}

func (h *HelmAdapter) Rollback(_ context.Context, cluster, namespace, name string, opts RollbackOptions) error {
	cfg, err := h.configuration(cluster, namespace)
	if err != nil {
		return err
	}
	client := action.NewRollback(cfg)
	client.Version = opts.Revision
	client.Wait = opts.Wait
	client.CleanupOnFail = opts.Atomic
	client.Timeout = opts.Timeout
	return client.Run(name)
}

func (h *HelmAdapter) Remove(_ context.Context, cluster, namespace, name string, opts RemoveOptions) error {
	cfg, err := h.configuration(cluster, namespace)
	if err != nil {
		return err
	}
	client := action.NewUninstall(cfg)
	client.KeepHistory = opts.KeepHistory
	client.Wait = opts.Wait
	client.Timeout = opts.Timeout
	_, err = client.Run(name)
	return err
}

func (h *HelmAdapter) configuration(cluster, namespace string) (*action.Configuration, error) {
	restConfig, err := h.clusters.RESTConfig(cluster)
	if err != nil {
		return nil, err
	}
	getter := &staticRESTClientGetter{config: rest.CopyConfig(restConfig), namespace: namespace}
	cfg := new(action.Configuration)
	if err := cfg.Init(getter, namespace, "secret", func(string, ...interface{}) {}); err != nil {
		return nil, fmt.Errorf("initialize helm client: %w", err)
	}
	return cfg, nil
}

type staticRESTClientGetter struct {
	config    *rest.Config
	namespace string
}

func (g *staticRESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return rest.CopyConfig(g.config), nil
}
func (g *staticRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	client, err := discovery.NewDiscoveryClientForConfig(g.config)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(client), nil
}
func (g *staticRESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	client, err := g.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(client), nil
}
func (g *staticRESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return &namespaceClientConfig{config: g.config, namespace: g.namespace}
}

// Helm consumes genericclioptions.RESTClientGetter; ConfigFlags already implements
// its kubeconfig loader, so delegate the loader without exposing credentials.
type namespaceClientConfig struct {
	config    *rest.Config
	namespace string
}

func (c *namespaceClientConfig) RawConfig() (clientcmdapi.Config, error) {
	return clientcmdapi.Config{}, nil
}
func (c *namespaceClientConfig) ClientConfig() (*rest.Config, error) {
	return rest.CopyConfig(c.config), nil
}
func (c *namespaceClientConfig) Namespace() (string, bool, error) {
	return c.namespace, c.namespace != "", nil
}
func (c *namespaceClientConfig) ConfigAccess() clientcmd.ConfigAccess { return nil }

func mapReleases(items []*release.Release) []Release {
	out := make([]Release, 0, len(items))
	for _, item := range items {
		if item != nil {
			out = append(out, mapRelease(item))
		}
	}
	return out
}
func mapRelease(item *release.Release) Release {
	r := Release{Name: item.Name, Namespace: item.Namespace, Revision: item.Version, Values: item.Config}
	if item.Info != nil {
		r.Status = item.Info.Status.String()
		r.UpdatedAt = item.Info.LastDeployed.Time
		r.Notes = item.Info.Notes
		r.Resources = parseManifest(item.Manifest)
	}
	if item.Chart != nil && item.Chart.Metadata != nil {
		r.Chart = item.Chart.Metadata.Name
		r.ChartVersion = item.Chart.Metadata.Version
		r.AppVersion = item.Chart.Metadata.AppVersion
	}
	return r
}

func parseManifest(manifest string) []ResourceRef {
	parts := strings.Split(manifest, "---")
	refs := make([]ResourceRef, 0, len(parts))
	for _, part := range parts {
		var header struct {
			APIVersion string `yaml:"apiVersion"`
			Kind       string `yaml:"kind"`
			Metadata   struct {
				Name      string `yaml:"name"`
				Namespace string `yaml:"namespace"`
			} `yaml:"metadata"`
		}
		if err := yaml.Unmarshal([]byte(part), &header); err == nil && header.Kind != "" && header.Metadata.Name != "" {
			refs = append(refs, ResourceRef{APIVersion: header.APIVersion, Kind: header.Kind, Namespace: header.Metadata.Namespace, Name: header.Metadata.Name})
		}
	}
	return refs
}
