package handler

import (
	"path/filepath"
	"testing"

	"github.com/zoobzio/jack/config"
)

// testRegistry returns an empty registry backed by a temp file.
func testRegistry(t *testing.T) *config.Registry {
	t.Helper()
	reg, err := config.NewRegistry(filepath.Join(t.TempDir(), "registry.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestResolveBothProvided(t *testing.T) {
	// Both given: resolve returns them without consulting the registry.
	agent, repo, err := resolve(testRegistry(t), "alex", "jack")
	if err != nil {
		t.Fatalf("resolve returned error: %v", err)
	}
	if agent != "alex" || repo != "jack" {
		t.Errorf("resolve = (%q, %q), want (alex, jack)", agent, repo)
	}
}

func TestResolveEmptyRegistry(t *testing.T) {
	if _, _, err := resolve(testRegistry(t), "", ""); err == nil {
		t.Fatal("resolve with no agents = nil error, want error")
	}
}

func TestResolveSingleAgentAndRepo(t *testing.T) {
	reg := testRegistry(t)
	reg.Add("alex", "jack", "https://host/u/jack.git")

	// A sole agent and a sole repo resolve automatically, no prompt.
	agent, repo, err := resolve(reg, "", "")
	if err != nil {
		t.Fatalf("resolve returned error: %v", err)
	}
	if agent != "alex" || repo != "jack" {
		t.Errorf("resolve = (%q, %q), want (alex, jack)", agent, repo)
	}
}

func TestResolveAgentWithoutRepos(t *testing.T) {
	reg := testRegistry(t)
	reg.Add("alex", "jack", "https://host/u/jack.git")

	// "bob" is provided but has no cloned repos.
	if _, _, err := resolve(reg, "bob", ""); err == nil {
		t.Fatal("resolve for an agent with no repos = nil error, want error")
	}
}
