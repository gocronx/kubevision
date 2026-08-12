package registry

import "testing"

func TestParseReference(t *testing.T) {
	tests := []struct {
		name, input string
		want        Reference
		invalid     bool
	}{
		{"docker shorthand", "nginx", Reference{Registry: "docker.io", Repository: "library/nginx"}, false},
		{"docker namespace and tag", "team/app:v1", Reference{Registry: "docker.io", Repository: "team/app", Tag: "v1"}, false},
		{"registry port", "registry.example:5443/team/app:edge", Reference{Registry: "registry.example:5443", Repository: "team/app", Tag: "edge"}, false},
		{"digest", "registry.example/team/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Reference{Registry: "registry.example", Repository: "team/app", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, false},
		{"tag and digest", "nginx:1@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Reference{Registry: "docker.io", Repository: "library/nginx", Tag: "1", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, false},
		{"reject url", "https://registry.example/team/app", Reference{}, true},
		{"reject uppercase repository", "registry.example/Team/app", Reference{}, true},
		{"reject short digest", "nginx@sha256:1234", Reference{}, true},
		{"reject empty component", "registry.example/team//app", Reference{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReference(tt.input)
			if tt.invalid {
				if err == nil {
					t.Fatalf("expected invalid reference, got %#v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}
