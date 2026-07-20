package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// StarterConfig carries the values used to seed a fresh config tree: the first
// agent's name and the git/GitHub identity to write into its profile. Empty
// identity fields are rendered as editable placeholders, so `jack init` can
// produce a usable file even when no global git identity is set.
type StarterConfig struct {
	Agent      string // first profile's agent name (validated by the caller)
	GitName    string // git user.name, or "" for a placeholder
	GitEmail   string // git user.email, or "" for a placeholder
	GitHubUser string // github login, or "" for a placeholder
}

// Render returns the body of a starter config.yaml: a commented template with
// one profile filled in from the identity fields (falling back to placeholders
// the user is expected to edit). It mirrors the documented config shape so the
// generated file doubles as reference.
func (s StarterConfig) Render() []byte {
	name, email, user := s.GitName, s.GitEmail, s.GitHubUser
	if name == "" {
		name = "Your Name"
	}
	if email == "" {
		email = "you@example.com"
	}
	if user == "" {
		user = "your-github-user"
	}

	return []byte(fmt.Sprintf(`# jack configuration — https://github.com/zoobzio/jack
#
# Top-level defaults, applied to any profile that doesn't set its own.
model: claude-opus-4-8          # ANTHROPIC_MODEL for the agent's claude
permission: acceptEdits         # default | acceptEdits | bypassPermissions

# Optional mTLS identity for agents, bootstrapped inside the container via
# step-cli. Uncomment and fill in to give agents certificate identity.
# ca:
#   url: https://ca.internal:9000
#   fingerprint: <root-ca-fingerprint>
#   provisioner: jack

# At least one profile is required. The map key is the agent name (no '-').
profiles:
  %s:
    git:
      name: %s
      email: %s
    github:
      user: %s
    # model: claude-opus-4-8         # optional per-agent override
    # permission: bypassPermissions  # optional per-agent override
    # repos:                         # optional supporting repos, mounted at /repos/<name>
    #   - https://github.com/you/lib
`, s.Agent, name, email, user))
}

// soul returns the starter CLAUDE.md written into the agent's config directory —
// a minimal, editable "soul" so the agent's config dir exists (clone requires
// it) and Claude Code has something to inherit.
func (s StarterConfig) soul() []byte {
	return []byte(fmt.Sprintf("# %s\n\nYou are %s, an agent operated through jack.\n", s.Agent, s.Agent))
}

// ScaffoldResult reports what Scaffold created, so the caller can tell the user
// what is new versus what was left untouched.
type ScaffoldResult struct {
	ConfigPath  string
	AgentDir    string
	WroteConfig bool // false if config.yaml already existed and was preserved
	WroteSoul   bool // false if the agent's CLAUDE.md already existed
}

// Scaffold lays out the config and data trees jack needs, seeded from sc. It
// creates ConfigDir with its agents/<agent> and projects children and DataDir,
// then writes a starter config.yaml and agents/<agent>/CLAUDE.md — but only when
// those files are absent, so re-running init never clobbers hand-edited config.
func (e *Env) Scaffold(sc StarterConfig) (ScaffoldResult, error) {
	agentDir := filepath.Join(e.ConfigDir, "agents", sc.Agent)
	dirs := []string{
		e.ConfigDir,
		agentDir,
		filepath.Join(e.ConfigDir, "projects"),
		e.DataDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return ScaffoldResult{}, fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	res := ScaffoldResult{ConfigPath: e.ConfigPath, AgentDir: agentDir}

	wrote, err := writeIfAbsent(e.ConfigPath, sc.Render())
	if err != nil {
		return ScaffoldResult{}, fmt.Errorf("writing config: %w", err)
	}
	res.WroteConfig = wrote

	soulPath := filepath.Join(agentDir, "CLAUDE.md")
	wrote, err = writeIfAbsent(soulPath, sc.soul())
	if err != nil {
		return ScaffoldResult{}, fmt.Errorf("writing agent soul: %w", err)
	}
	res.WroteSoul = wrote

	return res, nil
}

// writeIfAbsent writes content to path only when no file is already there,
// reporting whether it wrote. A pre-existing file is left exactly as-is.
func writeIfAbsent(path string, content []byte) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return false, err
	}
	return true, nil
}
