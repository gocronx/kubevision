package main

import (
	"runtime/debug"
	"testing"
)

func TestVersionOutput(t *testing.T) {
	oldVersion, oldCommit := version, commit
	t.Cleanup(func() {
		version, commit = oldVersion, oldCommit
	})

	version = "1.4.2+build.7"
	commit = "abc1234"

	if got, want := versionOutput(), "kubevision 1.4.2+build.7 (abc1234)"; got != want {
		t.Fatalf("versionOutput() = %q, want %q", got, want)
	}
}

func TestBuildMetadataUsesInjectedValues(t *testing.T) {
	oldVersion, oldCommit := version, commit
	t.Cleanup(func() {
		version, commit = oldVersion, oldCommit
	})

	version = "1.4.2"
	commit = "abc1234"
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v2.0.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "def5678"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	gotVersion, gotCommit := buildMetadata(info, true)
	if gotVersion != "1.4.2" || gotCommit != "abc1234" {
		t.Fatalf("buildMetadata() = (%q, %q), want (%q, %q)", gotVersion, gotCommit, "1.4.2", "abc1234")
	}
}

func TestBuildMetadataFallsBackToBuildInfo(t *testing.T) {
	oldVersion, oldCommit := version, commit
	t.Cleanup(func() {
		version, commit = oldVersion, oldCommit
	})

	version = "dev"
	commit = "unknown"
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v1.4.2"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "def5678"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	gotVersion, gotCommit := buildMetadata(info, true)
	if gotVersion != "1.4.2" || gotCommit != "def5678-dirty" {
		t.Fatalf("buildMetadata() = (%q, %q), want (%q, %q)", gotVersion, gotCommit, "1.4.2", "def5678-dirty")
	}
}

func TestBuildMetadataKeepsDevVersionForPseudoVersion(t *testing.T) {
	oldVersion, oldCommit := version, commit
	t.Cleanup(func() {
		version, commit = oldVersion, oldCommit
	})

	version = "dev"
	commit = "unknown"
	info := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.0.0-20260812192133-0ad35fa3b8c4+dirty"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "0ad35fa3b8c41bfb864a2129cd1a251042822b5a"},
			{Key: "vcs.modified", Value: "true"},
		},
	}

	gotVersion, gotCommit := buildMetadata(info, true)
	if gotVersion != "dev" || gotCommit != "0ad35fa3b8c41bfb864a2129cd1a251042822b5a-dirty" {
		t.Fatalf(
			"buildMetadata() = (%q, %q), want (%q, %q)",
			gotVersion,
			gotCommit,
			"dev",
			"0ad35fa3b8c41bfb864a2129cd1a251042822b5a-dirty",
		)
	}
}
