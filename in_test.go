//go:build testing

package jack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func stubRegistry(entries ...RegistryEntry) RegistryLoader {
	return func() (*Registry, error) {
		return &Registry{Projects: entries}, nil
	}
}

func stubAgentSelector(agent string) AgentSelector {
	return func(_ []string) (string, error) { return agent, nil }
}

func stubProjectSelector(project string) ProjectSelector {
	return func(_ string, _ []string) (string, error) { return project, nil }
}

var failSelector AgentSelector = func(_ []string) (string, error) {
	return "", nil
}

var failProjectSelector ProjectSelector = func(_ string, _ []string) (string, error) {
	return "", nil
}

var noopProvisioner BoardProvisioner = func(_, _ string) error { return nil }

func TestRunInEmptyRegistry(t *testing.T) {
	err := runIn("", "", stubRegistry(), failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher, noopAdder, noopDecrypter, noopProvisioner)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "no projects cloned"), true)
}

func TestRunInNoProjectsForAgent(t *testing.T) {
	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})
	err := runIn("red", "", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher, noopAdder, noopDecrypter, noopProvisioner)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "no projects cloned for agent"), true)
}

func TestRunInAttachesExistingSession(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var attachedName string
	attacher := func(name string) error {
		attachedName = name
		return nil
	}

	err := runIn("blue", "vicky", reg, failSelector, failProjectSelector,
		existsChecker, noopCreator, attacher, noopAdder, noopDecrypter, noopProvisioner)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, attachedName, "blue-vicky")
}

func TestRunInCreatesAndAttaches(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var createdName, attachedName string
	creator := func(name, dir, shellCmd string) error {
		createdName = name
		return nil
	}
	attacher := func(name string) error {
		attachedName = name
		return nil
	}

	err := runIn("blue", "vicky", reg, failSelector, failProjectSelector,
		noopChecker, creator, attacher, noopAdder, noopDecrypter, noopProvisioner)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, createdName, "blue-vicky")
	jtesting.AssertEqual(t, attachedName, "blue-vicky")
}

func TestRunInAutoSelectsSingleAgent(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var attachedName string
	attacher := func(name string) error {
		attachedName = name
		return nil
	}

	// No agent or project specified — should auto-select the only agent and project.
	err := runIn("", "", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, attacher, noopAdder, noopDecrypter, noopProvisioner)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, attachedName, "blue-vicky")
}

func TestRunInPromptsForAgent(t *testing.T) {
	newTestConfig()
	cfg.Profiles["red"] = Profile{Git: GitConfig{Name: "Red"}}
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(
		RegistryEntry{Agent: "blue", Repo: "vicky"},
		RegistryEntry{Agent: "red", Repo: "flux"},
	)

	var selectedAgent string
	agentSel := func(agents []string) (string, error) {
		selectedAgent = "red"
		return "red", nil
	}

	err := runIn("", "", reg, agentSel, failProjectSelector,
		noopChecker, noopCreator, noopAttacher, noopAdder, noopDecrypter, noopProvisioner)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, selectedAgent, "red")
}

func TestRunInPromptsForProject(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(
		RegistryEntry{Agent: "blue", Repo: "vicky"},
		RegistryEntry{Agent: "blue", Repo: "flux"},
	)

	var selectedProject string
	projSel := func(_ string, _ []string) (string, error) {
		selectedProject = "flux"
		return "flux", nil
	}

	err := runIn("blue", "", reg, failSelector, projSel,
		noopChecker, noopCreator, noopAttacher, noopAdder, noopDecrypter, noopProvisioner)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, selectedProject, "flux")
}

func TestRunInDecryptsToken(t *testing.T) {
	newTestConfig()
	dir := t.TempDir()
	env = Env{DataDir: dir, ConfigDir: t.TempDir()}

	// Create project dir with token.
	projDir := filepath.Join(dir, "blue", "vicky")
	jackDir := filepath.Join(projDir, ".jack")
	_ = os.MkdirAll(jackDir, 0o750)
	_ = os.WriteFile(filepath.Join(jackDir, "token.age"), []byte("encrypted"), 0o600)

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	creator := func(_, _, _ string) error { return nil }
	decrypter := func(_, _ string) (string, error) {
		return "tok_decrypted", nil
	}

	err := runIn("blue", "vicky", reg, failSelector, failProjectSelector,
		noopChecker, creator, noopAttacher, noopAdder, decrypter, noopProvisioner)
	jtesting.AssertNoError(t, err)

	// The session script is written to .jack/session.sh — verify the token is in it.
	scriptPath := filepath.Join(projDir, ".jack", "session.sh")
	scriptContent, err := os.ReadFile(scriptPath)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, strings.Contains(string(scriptContent), "export JACK_MSG_TOKEN=tok_decrypted"), true)
}

func TestRunInDecryptsGHToken(t *testing.T) {
	newTestConfig()
	dir := t.TempDir()
	configDir := t.TempDir()
	env = Env{DataDir: dir, ConfigDir: configDir}

	// Create project dir.
	projDir := filepath.Join(dir, "blue", "vicky")
	jackDir := filepath.Join(projDir, ".jack")
	_ = os.MkdirAll(jackDir, 0o750)

	// Create GitHub token file at the agent-level path.
	ghTokenDir := filepath.Join(configDir, "agents", "blue")
	_ = os.MkdirAll(ghTokenDir, 0o750)
	_ = os.WriteFile(filepath.Join(ghTokenDir, ".github-token.age"), []byte("encrypted"), 0o600)

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	creator := func(_, _, _ string) error { return nil }
	decrypter := func(_, _ string) (string, error) {
		return "ghp_decrypted", nil
	}

	err := runIn("blue", "vicky", reg, failSelector, failProjectSelector,
		noopChecker, creator, noopAttacher, noopAdder, decrypter, noopProvisioner)
	jtesting.AssertNoError(t, err)

	// Verify GH_TOKEN is in the session script.
	scriptPath := filepath.Join(projDir, ".jack", "session.sh")
	scriptContent, err := os.ReadFile(scriptPath)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, strings.Contains(string(scriptContent), "export GH_TOKEN=ghp_decrypted"), true)
}

func TestRunInUnknownAgentProfile(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "unknown", Repo: "vicky"})

	err := runIn("unknown", "vicky", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher, noopAdder, noopDecrypter, noopProvisioner)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "unknown agent"), true)
}

func TestBoardAutoJoinMatchEmpty(t *testing.T) {
	cfg = Config{}
	jtesting.AssertEqual(t, boardAutoJoinMatch("anything"), true)
}

func TestBoardAutoJoinMatchPattern(t *testing.T) {
	cfg = Config{Matrix: MatrixConfig{BoardAutoJoin: "^(blue|red)$"}}
	jtesting.AssertEqual(t, boardAutoJoinMatch("blue"), true)
	jtesting.AssertEqual(t, boardAutoJoinMatch("red"), true)
	jtesting.AssertEqual(t, boardAutoJoinMatch("green"), false)
}

func TestBoardAutoJoinMatchInvalidRegex(t *testing.T) {
	cfg = Config{Matrix: MatrixConfig{BoardAutoJoin: "[invalid"}}
	jtesting.AssertEqual(t, boardAutoJoinMatch("blue"), false)
}

func TestRunInSkipsBoardWhenAutoJoinNoMatch(t *testing.T) {
	newTestConfig()
	cfg.Matrix.BoardAutoJoin = "^red$"
	dir := t.TempDir()
	env = Env{DataDir: dir, ConfigDir: t.TempDir()}

	projDir := filepath.Join(dir, "blue", "vicky")
	jackDir := filepath.Join(projDir, ".jack")
	_ = os.MkdirAll(jackDir, 0o750)
	_ = os.WriteFile(filepath.Join(jackDir, "token.age"), []byte("encrypted"), 0o600)

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	provisioned := false
	provisioner := func(_, _ string) error {
		provisioned = true
		return nil
	}
	decrypter := func(_, _ string) (string, error) {
		return "tok_session", nil
	}

	err := runIn("blue", "vicky", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher, noopAdder, decrypter, provisioner)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, provisioned, false)
}

func TestRunInProvisionsBoardWithToken(t *testing.T) {
	newTestConfig()
	dir := t.TempDir()
	env = Env{DataDir: dir, ConfigDir: t.TempDir()}

	projDir := filepath.Join(dir, "blue", "vicky")
	jackDir := filepath.Join(projDir, ".jack")
	_ = os.MkdirAll(jackDir, 0o750)
	_ = os.WriteFile(filepath.Join(jackDir, "token.age"), []byte("encrypted"), 0o600)

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var provisionedToken, provisionedName string
	provisioner := func(token, name string) error {
		provisionedToken = token
		provisionedName = name
		return nil
	}
	decrypter := func(_, _ string) (string, error) {
		return "tok_session", nil
	}

	err := runIn("blue", "vicky", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher, noopAdder, decrypter, provisioner)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, provisionedToken, "tok_session")
	jtesting.AssertEqual(t, provisionedName, "blue-vicky")
}
