package domain

import "testing"

func TestNewIdentityMainClone(t *testing.T) {
	id, err := NewIdentity("claude", "repo")
	if err != nil {
		t.Fatalf("NewIdentity() unexpected error: %v", err)
	}
	if got, want := id.Container, "jack-claude-repo"; got != want {
		t.Errorf("Container = %q, want %q", got, want)
	}
	if got, want := id.Session, "claude-repo"; got != want {
		t.Errorf("Session = %q, want %q", got, want)
	}
	if got, want := id.Agent(), Agent("claude"); got != want {
		t.Errorf("Agent() = %q, want %q", got, want)
	}
	if got, want := id.Repo(), Repo("repo"); got != want {
		t.Errorf("Repo() = %q, want %q", got, want)
	}
	if got, want := id.RepoPath(), "/root/workspace/repo"; got != want {
		t.Errorf("RepoPath() = %q, want %q", got, want)
	}
	if got, want := id.ToolsVolume(), "jack-claude-repo-tools"; got != want {
		t.Errorf("ToolsVolume() = %q, want %q", got, want)
	}
}

func TestNewIdentityErrors(t *testing.T) {
	tests := []struct {
		name  string
		agent Agent
		repo  Repo
	}{
		{name: "invalid agent (empty)", agent: "", repo: "repo"},
		{name: "invalid agent (hyphen)", agent: "foo-bar", repo: "repo"},
		{name: "invalid repo (empty)", agent: "claude", repo: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewIdentity(tt.agent, tt.repo); err == nil {
				t.Fatalf("NewIdentity() error = nil, want error")
			}
		})
	}
}

func TestContainerHome(t *testing.T) {
	if ContainerHome != "/root" {
		t.Fatalf("ContainerHome = %q, want %q", ContainerHome, "/root")
	}
}
