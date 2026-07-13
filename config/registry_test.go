package config

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/zoobzio/jack/domain"
)

func TestRegistryAddAndFind(t *testing.T) {
	reg := &Registry{path: filepath.Join(t.TempDir(), "registry.yaml")}

	if got := reg.Find("alex", "foo"); got != nil {
		t.Fatalf("Find on empty registry = %+v, want nil", got)
	}

	reg.Add("alex", "foo", "https://github.com/x/foo")

	got := reg.Find("alex", "foo")
	if got == nil {
		t.Fatal("Find after Add = nil, want entry")
	}
	if got.Agent != "alex" || got.Repo != "foo" || got.URL != "https://github.com/x/foo" {
		t.Errorf("entry = %+v, want agent=alex repo=foo url set", got)
	}
	if got.ClonedAt.IsZero() {
		t.Error("ClonedAt was not set")
	}
}

func TestRegistryAddReplacesExisting(t *testing.T) {
	reg := &Registry{path: filepath.Join(t.TempDir(), "registry.yaml")}

	reg.Add("alex", "foo", "url-old")
	reg.Add("alex", "foo", "url-new")

	if len(reg.Projects) != 1 {
		t.Fatalf("expected 1 entry after replace, got %d: %+v", len(reg.Projects), reg.Projects)
	}
	if got := reg.Find("alex", "foo"); got == nil || got.URL != "url-new" {
		t.Errorf("Find = %+v, want URL url-new", got)
	}
}

func TestRegistryRemove(t *testing.T) {
	reg := &Registry{path: filepath.Join(t.TempDir(), "registry.yaml")}

	reg.Add("alex", "foo", "u1")
	reg.Add("alex", "bar", "u2")
	reg.Remove("alex", "foo")

	if got := reg.Find("alex", "foo"); got != nil {
		t.Errorf("Find after Remove = %+v, want nil", got)
	}
	if got := reg.Find("alex", "bar"); got == nil {
		t.Error("Remove deleted the wrong entry")
	}
	if len(reg.Projects) != 1 {
		t.Errorf("expected 1 entry after Remove, got %d", len(reg.Projects))
	}

	// Removing a nonexistent entry is a no-op.
	reg.Remove("nobody", "nope")
	if len(reg.Projects) != 1 {
		t.Errorf("Remove of nonexistent changed length to %d", len(reg.Projects))
	}
}

func TestRegistryForAgent(t *testing.T) {
	reg := &Registry{path: filepath.Join(t.TempDir(), "registry.yaml")}

	reg.Add("alex", "zeta", "u1")
	reg.Add("alex", "alpha", "u2")
	reg.Add("bob", "middle", "u3")

	entries := reg.ForAgent("alex")
	if len(entries) != 2 {
		t.Fatalf("ForAgent(alex) len = %d, want 2", len(entries))
	}
	// Sorted by repo name.
	if entries[0].Repo != "alpha" || entries[1].Repo != "zeta" {
		t.Errorf("ForAgent(alex) repos = [%s %s], want [alpha zeta]", entries[0].Repo, entries[1].Repo)
	}
	for _, e := range entries {
		if e.Agent != "alex" {
			t.Errorf("ForAgent(alex) returned entry for agent %q", e.Agent)
		}
	}

	if got := reg.ForAgent("nobody"); got != nil {
		t.Errorf("ForAgent(nobody) = %+v, want nil", got)
	}
}

func TestRegistryAgents(t *testing.T) {
	reg := &Registry{path: filepath.Join(t.TempDir(), "registry.yaml")}

	reg.Add("charlie", "r1", "u")
	reg.Add("alex", "r2", "u")
	reg.Add("alex", "r3", "u")
	reg.Add("bob", "r4", "u")

	got := reg.Agents()
	want := []domain.Agent{"alex", "bob", "charlie"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Agents() = %v, want %v", got, want)
	}
}

func TestRegistryReposForAgent(t *testing.T) {
	reg := &Registry{path: filepath.Join(t.TempDir(), "registry.yaml")}

	reg.Add("alex", "zeta", "u")
	reg.Add("alex", "alpha", "u")
	reg.Add("bob", "other", "u")

	got := reg.ReposForAgent("alex")
	want := []domain.Repo{"alpha", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReposForAgent(alex) = %v, want %v", got, want)
	}
}

func TestRegistryRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "registry.yaml")

	// Missing file → empty registry.
	reg, err := NewRegistry(path)
	if err != nil {
		t.Fatalf("NewRegistry on missing file returned error: %v", err)
	}
	if len(reg.Projects) != 0 {
		t.Fatalf("expected empty registry, got %d entries", len(reg.Projects))
	}

	reg.Add("alex", "foo", "url-foo")
	reg.Add("bob", "bar", "url-bar")
	if err := reg.Save(); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	reloaded, err := NewRegistry(path)
	if err != nil {
		t.Fatalf("NewRegistry after Save returned error: %v", err)
	}
	if len(reloaded.Projects) != 2 {
		t.Fatalf("reloaded registry has %d entries, want 2", len(reloaded.Projects))
	}
	if got := reloaded.Find("alex", "foo"); got == nil || got.URL != "url-foo" {
		t.Errorf("reloaded Find(alex,foo) = %+v, want URL url-foo", got)
	}
	if got := reloaded.Find("bob", "bar"); got == nil || got.URL != "url-bar" {
		t.Errorf("reloaded Find(bob,bar) = %+v, want URL url-bar", got)
	}
}
