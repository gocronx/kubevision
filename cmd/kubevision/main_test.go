package main

import "testing"

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
