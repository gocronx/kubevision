package packages

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	bizerr "github.com/gocronx/kubevision/internal/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
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

type HelmAdapter struct {
	clusters RESTConfigProvider
	catalog  *Catalog
}

func NewHelmAdapter(clusters RESTConfigProvider) *HelmAdapter {
	return &HelmAdapter{clusters: clusters}
}

func (h *HelmAdapter) WithCatalog(catalog *Catalog) *HelmAdapter { h.catalog = catalog; return h }

func (h *HelmAdapter) loadChart(ctx context.Context, source ChartSource) (*chart.Chart, error) {
	if h.catalog != nil {
		return h.catalog.loadChart(ctx, source)
	}
	return loadRemoteChart(ctx, source)
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

func (h *HelmAdapter) Preview(ctx context.Context, operation, cluster string, opts ChangeOptions) (*Preview, error) {
	cfg, err := h.configuration(cluster, opts.Namespace)
	if err != nil {
		return nil, err
	}
	chrt, err := h.loadChart(ctx, opts.Source)
	if err != nil {
		return nil, err
	}
	item, err := renderChange(ctx, cfg, operation, chrt, opts)
	if err != nil {
		return nil, err
	}
	digest := changeDigest(chrt, item)
	manifest := releaseManifest(item)
	resources, risks := inspectManifest(manifest)
	result := &Preview{Chart: chrt.Metadata.Name, ChartVersion: chrt.Metadata.Version, AppVersion: chrt.Metadata.AppVersion, Digest: digest, Manifest: manifest, Resources: resources, Risks: risks}
	return result, nil
}

func (h *HelmAdapter) Install(ctx context.Context, cluster string, opts ChangeOptions) error {
	cfg, err := h.configuration(cluster, opts.Namespace)
	if err != nil {
		return err
	}
	chrt, err := h.loadChart(ctx, opts.Source)
	if err != nil {
		return err
	}
	preview, err := renderChange(ctx, cfg, "install", chrt, opts)
	if err != nil {
		return err
	}
	if changeDigest(chrt, preview) != opts.ExpectedDigest {
		return fmt.Errorf("rendered resources changed after preview")
	}
	client := action.NewInstall(cfg)
	configureInstall(client, opts, false)
	_, err = client.RunWithContext(ctx, chrt, opts.Values)
	return err
}

func (h *HelmAdapter) Upgrade(ctx context.Context, cluster string, opts ChangeOptions) error {
	cfg, err := h.configuration(cluster, opts.Namespace)
	if err != nil {
		return err
	}
	chrt, err := h.loadChart(ctx, opts.Source)
	if err != nil {
		return err
	}
	preview, err := renderChange(ctx, cfg, "upgrade", chrt, opts)
	if err != nil {
		return err
	}
	if changeDigest(chrt, preview) != opts.ExpectedDigest {
		return fmt.Errorf("rendered resources changed after preview")
	}
	client := action.NewUpgrade(cfg)
	configureUpgrade(client, opts, false)
	_, err = client.RunWithContext(ctx, opts.ReleaseName, chrt, opts.Values)
	return err
}

func renderChange(ctx context.Context, cfg *action.Configuration, operation string, chrt *chart.Chart, opts ChangeOptions) (*release.Release, error) {
	if operation == "install" {
		client := action.NewInstall(cfg)
		configureInstall(client, opts, true)
		return client.RunWithContext(ctx, chrt, opts.Values)
	}
	client := action.NewUpgrade(cfg)
	configureUpgrade(client, opts, true)
	return client.RunWithContext(ctx, opts.ReleaseName, chrt, opts.Values)
}

func releaseManifest(item *release.Release) string {
	parts := []string{item.Manifest}
	for _, hook := range item.Hooks {
		if hook != nil && strings.TrimSpace(hook.Manifest) != "" {
			parts = append(parts, hook.Manifest)
		}
	}
	return strings.Join(parts, "\n---\n")
}

func changeDigest(chrt *chart.Chart, item *release.Release) string {
	hash := sha256.New()
	hash.Write([]byte(chartDigest(chrt)))
	hash.Write([]byte{0})
	hash.Write([]byte(releaseManifest(item)))
	return hex.EncodeToString(hash.Sum(nil))
}

func configureInstall(client *action.Install, opts ChangeOptions, dryRun bool) {
	client.ReleaseName, client.Namespace = opts.ReleaseName, opts.Namespace
	client.CreateNamespace, client.Wait, client.Atomic, client.Timeout = opts.CreateNamespace, opts.Wait, opts.Atomic, opts.Timeout
	client.DryRun, client.HideSecret = dryRun, dryRun
	if dryRun {
		client.DryRunOption = "server"
	}
	client.IncludeCRDs = dryRun
}

func configureUpgrade(client *action.Upgrade, opts ChangeOptions, dryRun bool) {
	client.Namespace, client.Wait, client.Atomic, client.Timeout = opts.Namespace, opts.Wait, opts.Atomic, opts.Timeout
	client.DryRun, client.HideSecret = dryRun, dryRun
	if dryRun {
		client.DryRunOption = "server"
	}
	client.ResetThenReuseValues = true
}

func loadRemoteChart(ctx context.Context, source ChartSource) (*chart.Chart, error) {
	if err := validateChartSource(source); err != nil {
		return nil, bizerr.New(bizerr.CodeValidation, err.Error())
	}
	tempDir, err := os.MkdirTemp("", "kubevision-chart-*")
	if err != nil {
		return nil, fmt.Errorf("create chart workspace: %w", err)
	}
	defer os.RemoveAll(tempDir)
	if strings.HasPrefix(source.Chart, "oci://") {
		settings := cli.New()
		settings.RepositoryCache = filepath.Join(tempDir, "cache")
		client := action.NewInstall(new(action.Configuration))
		client.ChartPathOptions.Version = source.Version
		httpClient := chartHTTPClient(source.AllowPrivateNetwork)
		defer httpClient.CloseIdleConnections()
		options := []registry.ClientOption{registry.ClientOptWriter(io.Discard), registry.ClientOptCredentialsFile(filepath.Join(tempDir, "registry.json")), registry.ClientOptHTTPClient(httpClient)}
		if source.Username != "" || source.Password != "" {
			options = append(options, registry.ClientOptBasicAuth(source.Username, source.Password))
		}
		registryClient, registryErr := registry.NewClient(options...)
		if registryErr != nil {
			return nil, fmt.Errorf("initialize OCI client: %w", registryErr)
		}
		client.SetRegistryClient(registryClient)
		path, locateErr := client.ChartPathOptions.LocateChart(source.Chart, settings)
		if locateErr != nil {
			return nil, fmt.Errorf("locate chart: %w", locateErr)
		}
		loaded, loadErr := loader.Load(path)
		if loadErr != nil {
			return nil, fmt.Errorf("load chart: %w", loadErr)
		}
		if loaded.Metadata == nil {
			return nil, fmt.Errorf("chart metadata is missing")
		}
		if err := rejectClusterLookups(loaded); err != nil {
			return nil, err
		}
		return loaded, nil
	}
	chartURL := source.Chart
	if source.RepoURL != "" {
		base, _ := url.Parse(strings.TrimRight(source.RepoURL, "/") + "/")
		indexURL := base.ResolveReference(&url.URL{Path: "index.yaml"})
		indexData, fetchErr := fetchWithSource(ctx, indexURL.String(), 10<<20, source)
		if fetchErr != nil {
			return nil, fmt.Errorf("download repository index: %w", fetchErr)
		}
		var index repo.IndexFile
		if err := yaml.Unmarshal(indexData, &index); err != nil {
			return nil, fmt.Errorf("parse repository index: %w", err)
		}
		version, getErr := index.Get(source.Chart, source.Version)
		if getErr != nil || len(version.URLs) == 0 {
			return nil, fmt.Errorf("chart version not found in repository")
		}
		reference, parseErr := url.Parse(version.URLs[0])
		if parseErr != nil {
			return nil, fmt.Errorf("invalid chart URL in repository")
		}
		chartURL = base.ResolveReference(reference).String()
	}
	archive, err := fetchWithSource(ctx, chartURL, 50<<20, source)
	if err != nil {
		return nil, fmt.Errorf("download chart: %w", err)
	}
	loaded, err := loader.LoadArchive(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("load chart: %w", err)
	}
	if loaded.Metadata == nil {
		return nil, fmt.Errorf("chart metadata is missing")
	}
	if err := rejectClusterLookups(loaded); err != nil {
		return nil, err
	}
	return loaded, nil
}

func rejectClusterLookups(chrt *chart.Chart) error {
	for _, template := range chrt.Templates {
		fields := strings.FieldsFunc(string(template.Data), func(r rune) bool {
			return !(r == '_' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
		})
		for _, field := range fields {
			if field == "lookup" {
				return bizerr.New(bizerr.CodeValidation, "chart templates using Helm lookup are not supported for security reasons")
			}
		}
	}
	for _, dependency := range chrt.Dependencies() {
		if err := rejectClusterLookups(dependency); err != nil {
			return err
		}
	}
	return nil
}

func fetchPublic(ctx context.Context, rawURL string, limit int64) ([]byte, error) {
	return fetchWithSource(ctx, rawURL, limit, ChartSource{})
}

func fetchWithSource(ctx context.Context, rawURL string, limit int64, source ChartSource) ([]byte, error) {
	if err := validateRepositoryURL(rawURL, source.AllowPrivateNetwork); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if source.Username != "" || source.Password != "" {
		req.SetBasicAuth(source.Username, source.Password)
	}
	client := chartHTTPClient(source.AllowPrivateNetwork)
	defer client.CloseIdleConnections()
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download exceeds %d byte limit", limit)
	}
	return data, nil
}

func publicHTTPClient() *http.Client { return chartHTTPClient(false) }

func chartHTTPClient(allowPrivate bool) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{ForceAttemptHTTP2: true, MaxIdleConns: 20, MaxIdleConnsPerHost: 4, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 20 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil || len(addresses) == 0 {
			return nil, fmt.Errorf("host could not be resolved")
		}
		for _, candidate := range addresses {
			if allowPrivate || isPublicIP(candidate.IP) {
				return dialer.DialContext(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
			}
		}
		return nil, fmt.Errorf("private or local hosts are not allowed")
	}
	return &http.Client{Transport: &publicRoundTripper{transport: transport}, Timeout: 2 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		if err := validateRepositoryURL(req.URL.String(), allowPrivate); err != nil {
			return err
		}
		if len(via) > 0 && !sameOrigin(via[0].URL, req.URL) {
			req.Header.Del("Authorization")
		}
		return nil
	}}
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

type publicRoundTripper struct{ transport *http.Transport }

func (t *publicRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Scheme != "https" || request.URL.Hostname() == "" || request.URL.User != nil {
		return nil, fmt.Errorf("only credential-free HTTPS requests are allowed")
	}
	return t.transport.RoundTrip(request)
}

func (t *publicRoundTripper) CloseIdleConnections() { t.transport.CloseIdleConnections() }

func isPublicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast()
}

func validateChartSource(source ChartSource) error {
	chartRef, repoURL := strings.TrimSpace(source.Chart), strings.TrimSpace(source.RepoURL)
	if filepath.IsAbs(chartRef) || strings.HasPrefix(chartRef, ".") || strings.HasPrefix(chartRef, "file:") {
		return fmt.Errorf("local chart paths are not allowed")
	}
	if repoURL != "" {
		if strings.Contains(chartRef, "://") {
			return fmt.Errorf("chart must be a repository chart name when repoUrl is set")
		}
		if err := validateRepositoryURL(repoURL, source.AllowPrivateNetwork); err != nil {
			return fmt.Errorf("invalid repository URL: %w", err)
		}
		return nil
	}
	if strings.HasPrefix(chartRef, "oci://") {
		u, err := url.Parse(chartRef)
		if err != nil || u.Hostname() == "" || u.User != nil {
			return fmt.Errorf("invalid OCI chart reference")
		}
		if source.AllowPrivateNetwork {
			return nil
		}
		return validatePublicHost(u.Hostname())
	}
	return validateRepositoryURL(chartRef, source.AllowPrivateNetwork)
}

func validateRepositoryURL(raw string, allowPrivate bool) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return fmt.Errorf("only absolute HTTPS URLs are allowed")
	}
	if u.User != nil {
		return fmt.Errorf("credentials in URLs are not allowed")
	}
	if allowPrivate {
		return nil
	}
	return validatePublicHost(u.Hostname())
}

func validateOCIBase(raw string, allowPrivate bool) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "oci" || u.Hostname() == "" || u.User != nil {
		return fmt.Errorf("a valid oci:// URL is required")
	}
	if allowPrivate {
		return nil
	}
	return validatePublicHost(u.Hostname())
}

func validatePublicHTTPS(raw string) error {
	return validateRepositoryURL(raw, false)
}

func validatePublicHost(host string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(addresses) == 0 {
		return fmt.Errorf("host could not be resolved")
	}
	for _, address := range addresses {
		if !isPublicIP(address.IP) {
			return fmt.Errorf("private or local hosts are not allowed")
		}
	}
	return nil
}

func chartDigest(chrt *chart.Chart) string {
	type entry struct {
		Name string
		Data []byte
	}
	entries := []entry{}
	var collect func(*chart.Chart, string)
	collect = func(c *chart.Chart, prefix string) {
		metadata, _ := json.Marshal(c.Metadata)
		values, _ := json.Marshal(c.Values)
		entries = append(entries, entry{prefix + "Chart.json", metadata}, entry{prefix + "values.json", values})
		for _, f := range append(append(c.Raw, c.Templates...), c.Files...) {
			entries = append(entries, entry{prefix + f.Name, f.Data})
		}
		for _, dep := range c.Dependencies() {
			collect(dep, prefix+dep.Name()+"/")
		}
	}
	collect(chrt, "")
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	hash := sha256.New()
	for _, item := range entries {
		hash.Write([]byte(item.Name))
		hash.Write([]byte{0})
		hash.Write(item.Data)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func inspectManifest(manifest string) ([]ResourceRef, []Risk) {
	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(manifest), 4096)
	refs, risks := []ResourceRef{}, []Risk{}
	for {
		var object unstructured.Unstructured
		if err := decoder.Decode(&object); err != nil {
			if err != io.EOF {
				risks = append(risks, Risk{Level: "critical", Code: "manifest-parse", Message: "rendered manifest could not be fully inspected"})
			}
			break
		}
		if object.GetKind() == "" || object.GetName() == "" {
			continue
		}
		ref := ResourceRef{APIVersion: object.GetAPIVersion(), Kind: object.GetKind(), Namespace: object.GetNamespace(), Name: object.GetName()}
		refs = append(refs, ref)
		resource := object.GetKind() + "/" + object.GetName()
		switch object.GetKind() {
		case "CustomResourceDefinition", "ClusterRole", "ClusterRoleBinding", "Role", "RoleBinding", "MutatingWebhookConfiguration", "ValidatingWebhookConfiguration", "Namespace":
			risks = append(risks, Risk{Level: "critical", Code: "security-sensitive-resource", Message: "creates or changes a security-sensitive resource", Resource: resource})
		}
		podSpec, found, _ := unstructured.NestedMap(object.Object, workloadPodSpecPath(object.GetKind())...)
		if found {
			risks = append(risks, inspectPodSpec(podSpec, resource)...)
		}
	}
	return refs, risks
}

func workloadPodSpecPath(kind string) []string {
	switch kind {
	case "Pod":
		return []string{"spec"}
	case "CronJob":
		return []string{"spec", "jobTemplate", "spec", "template", "spec"}
	case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet", "Job":
		return []string{"spec", "template", "spec"}
	}
	return []string{"_missing"}
}

func inspectPodSpec(spec map[string]interface{}, resource string) []Risk {
	risks := []Risk{}
	for _, field := range []string{"hostNetwork", "hostPID", "hostIPC"} {
		if enabled, _, _ := unstructured.NestedBool(spec, field); enabled {
			risks = append(risks, Risk{Level: "critical", Code: "host-access", Message: field + " is enabled", Resource: resource})
		}
	}
	volumes, _, _ := unstructured.NestedSlice(spec, "volumes")
	for _, raw := range volumes {
		if volume, ok := raw.(map[string]interface{}); ok {
			if _, exists := volume["hostPath"]; exists {
				risks = append(risks, Risk{Level: "critical", Code: "host-path", Message: "mounts a hostPath volume", Resource: resource})
			}
		}
	}
	for _, group := range []string{"containers", "initContainers", "ephemeralContainers"} {
		containers, _, _ := unstructured.NestedSlice(spec, group)
		for _, raw := range containers {
			container, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			privileged, _, _ := unstructured.NestedBool(container, "securityContext", "privileged")
			if privileged {
				risks = append(risks, Risk{Level: "critical", Code: "privileged-container", Message: "runs a privileged container", Resource: resource})
			}
			allowEscalation, _, _ := unstructured.NestedBool(container, "securityContext", "allowPrivilegeEscalation")
			if allowEscalation {
				risks = append(risks, Risk{Level: "critical", Code: "privilege-escalation", Message: "allows process privilege escalation", Resource: resource})
			}
			runAsUser, foundUser, _ := unstructured.NestedInt64(container, "securityContext", "runAsUser")
			if foundUser && runAsUser == 0 {
				risks = append(risks, Risk{Level: "critical", Code: "root-container", Message: "explicitly runs as root", Resource: resource})
			}
			capabilities, foundCaps, _ := unstructured.NestedStringSlice(container, "securityContext", "capabilities", "add")
			if foundCaps && len(capabilities) > 0 {
				risks = append(risks, Risk{Level: "critical", Code: "linux-capabilities", Message: "adds Linux capabilities", Resource: resource})
			}
			ports, _, _ := unstructured.NestedSlice(container, "ports")
			for _, rawPort := range ports {
				if port, ok := rawPort.(map[string]interface{}); ok {
					if hostPort, found, _ := unstructured.NestedInt64(port, "hostPort"); found && hostPort > 0 {
						risks = append(risks, Risk{Level: "critical", Code: "host-port", Message: "binds a port on the host", Resource: resource})
						break
					}
				}
			}
		}
	}
	return risks
}

func redactManifest(manifest string) string {
	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(manifest), 4096)
	documents := []string{}
	for {
		var object map[string]interface{}
		if err := decoder.Decode(&object); err != nil {
			break
		}
		if len(object) == 0 {
			continue
		}
		if kind, _ := object["kind"].(string); kind == "Secret" {
			object["data"] = map[string]interface{}{"redacted": "[REDACTED]"}
			object["stringData"] = map[string]interface{}{"redacted": "[REDACTED]"}
		}
		encoded, err := yaml.Marshal(object)
		if err == nil {
			documents = append(documents, string(encoded))
		}
	}
	return strings.Join(documents, "---\n")
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
