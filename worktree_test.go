//go:build testing

package jack

import (
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestWorktreeHashDeterministic(t *testing.T) {
	h1 := WorktreeHash("feature-login")
	h2 := WorktreeHash("feature-login")
	jtesting.AssertEqual(t, h1, h2)
}

func TestWorktreeHashLength(t *testing.T) {
	h := WorktreeHash("my-branch")
	jtesting.AssertEqual(t, len(h), 8)
}

func TestWorktreeHashDifferentInputs(t *testing.T) {
	h1 := WorktreeHash("feature-login")
	h2 := WorktreeHash("feature-signup")
	jtesting.AssertEqual(t, h1 != h2, true)
}

func TestWorktreeDir(t *testing.T) {
	hash := WorktreeHash("feature-login")
	got := WorktreeDir("vicky", "feature-login")
	want := "vicky-" + hash
	jtesting.AssertEqual(t, got, want)
}

func TestWorktreeDirFormat(t *testing.T) {
	got := WorktreeDir("flux", "main")
	hash := WorktreeHash("main")
	jtesting.AssertEqual(t, got, "flux-"+hash)
}

func TestWorktreeContainerPath(t *testing.T) {
	hash := WorktreeHash("feature-login")
	got := WorktreeContainerPath("vicky", "feature-login")
	want := "/root/workspace/vicky-" + hash
	jtesting.AssertEqual(t, got, want)
}

func TestWorktreeContainerPathFormat(t *testing.T) {
	got := WorktreeContainerPath("flux", "dev")
	hash := WorktreeHash("dev")
	jtesting.AssertEqual(t, got, "/root/workspace/flux-"+hash)
}
