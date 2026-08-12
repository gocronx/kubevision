package registry

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	namePartPattern = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*$`)
	tagPattern      = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)
	digestPattern   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_+.-]*:[A-Fa-f0-9]{32,}$`)
)

// Reference is the normalized location of an OCI image.
type Reference struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`
	Tag        string `json:"tag,omitempty"`
	Digest     string `json:"digest,omitempty"`
}

// ParseReference accepts familiar Docker shorthand but never accepts a URL.
func ParseReference(value string) (Reference, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "://") || strings.ContainsAny(value, " \t\r\n?#") {
		return Reference{}, fmt.Errorf("invalid image reference")
	}

	var digest string
	if before, after, ok := strings.Cut(value, "@"); ok {
		if strings.Contains(after, "@") || !digestPattern.MatchString(after) {
			return Reference{}, fmt.Errorf("invalid image digest")
		}
		value, digest = before, after
	}

	var tag string
	lastSlash, lastColon := strings.LastIndex(value, "/"), strings.LastIndex(value, ":")
	if lastColon > lastSlash {
		tag = value[lastColon+1:]
		value = value[:lastColon]
		if !tagPattern.MatchString(tag) {
			return Reference{}, fmt.Errorf("invalid image tag")
		}
	}

	parts := strings.Split(value, "/")
	if len(parts) == 0 {
		return Reference{}, fmt.Errorf("invalid image repository")
	}
	registryHost := "docker.io"
	if len(parts) > 1 && isRegistryComponent(parts[0]) {
		registryHost = strings.ToLower(parts[0])
		parts = parts[1:]
	}
	if registryHost == "index.docker.io" || registryHost == "registry-1.docker.io" {
		registryHost = "docker.io"
	}
	if len(parts) == 1 && registryHost == "docker.io" {
		parts = append([]string{"library"}, parts...)
	}
	if len(parts) == 0 {
		return Reference{}, fmt.Errorf("missing image repository")
	}
	for _, part := range parts {
		if len(part) == 0 || len(part) > 255 || !namePartPattern.MatchString(part) {
			return Reference{}, fmt.Errorf("invalid image repository")
		}
	}
	if len(registryHost) > 255 || strings.HasPrefix(registryHost, ".") || strings.HasSuffix(registryHost, ".") {
		return Reference{}, fmt.Errorf("invalid registry")
	}

	return Reference{Registry: registryHost, Repository: strings.Join(parts, "/"), Tag: tag, Digest: digest}, nil
}

func isRegistryComponent(value string) bool {
	return value == "localhost" || strings.Contains(value, ".") || strings.Contains(value, ":")
}
