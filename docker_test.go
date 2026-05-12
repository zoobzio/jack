//go:build testing

package jack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestContainerName(t *testing.T) {
	jtesting.AssertEqual(t, ContainerName("blue", "vicky"), "jack-blue-vicky")
	jtesting.AssertEqual(t, ContainerName("red", "flux"), "jack-red-flux")
}

func TestSessionMountsBase(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}
	cfg = Config{}

	profile := Profile{}
	repoDir := t.TempDir()

	mounts := SessionMounts(profile, "blue", repoDir)

	home, _ := os.UserHomeDir()
	// .claude, .claude.json, agent .claude/, repo = 4
	jtesting.AssertEqual(t, len(mounts), 4)
	jtesting.AssertEqual(t, mounts[0].Source, filepath.Join(home, ".claude"))
	jtesting.AssertEqual(t, mounts[0].Target, "/home/jack/.claude")
	jtesting.AssertEqual(t, mounts[1].Source, filepath.Join(home, ".claude.json"))
	jtesting.AssertEqual(t, mounts[1].Target, "/home/jack/.claude.json")
	jtesting.AssertEqual(t, mounts[2].Target, "/home/jack/workspace/.claude")
	jtesting.AssertEqual(t, mounts[2].ReadOnly, true)
	jtesting.AssertEqual(t, mounts[3].Source, repoDir)
	jtesting.AssertEqual(t, mounts[3].Target, "/home/jack/workspace/repo")
	jtesting.AssertEqual(t, mounts[3].ReadOnly, false)
}

func TestSessionMountsWithCert(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}
	cfg = Config{}

	// Create the agent cert and key files so hasCert returns true.
	agentDir := filepath.Join(configDir, "agents", "blue")
	_ = os.MkdirAll(agentDir, 0o750)
	_ = os.WriteFile(filepath.Join(agentDir, "cert.pem"), []byte("cert"), 0o600)
	_ = os.WriteFile(filepath.Join(agentDir, "key.pem"), []byte("key"), 0o600)

	profile := Profile{}
	repoDir := t.TempDir()

	mounts := SessionMounts(profile, "blue", repoDir)

	// base 4 + cert + key = 6
	jtesting.AssertEqual(t, len(mounts), 6)
	jtesting.AssertEqual(t, mounts[4].Target, "/home/jack/.jack/cert.pem")
	jtesting.AssertEqual(t, mounts[4].ReadOnly, true)
	jtesting.AssertEqual(t, mounts[5].Target, "/home/jack/.jack/key.pem")
	jtesting.AssertEqual(t, mounts[5].ReadOnly, true)
}

func TestSessionMountsWithCARoot(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}

	caPath := filepath.Join(configDir, "ca.pem")
	_ = os.WriteFile(caPath, []byte("ca-cert"), 0o600)
	cfg = Config{CA: CAConfig{Root: caPath}}

	profile := Profile{}
	repoDir := t.TempDir()

	mounts := SessionMounts(profile, "blue", repoDir)

	// base 4 + ca root = 5
	jtesting.AssertEqual(t, len(mounts), 5)
	jtesting.AssertEqual(t, mounts[4].Source, caPath)
	jtesting.AssertEqual(t, mounts[4].Target, "/home/jack/.jack/ca.pem")
	jtesting.AssertEqual(t, mounts[4].ReadOnly, true)
}

func TestSessionMountsWithSupportingRepos(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}
	cfg = Config{}

	supportDir := filepath.Join(dataDir, "blue", "vicky")
	_ = os.MkdirAll(supportDir, 0o750)

	profile := Profile{
		Repos: []string{"git@github.com:zoobzio/vicky.git"},
	}
	repoDir := t.TempDir()

	mounts := SessionMounts(profile, "blue", repoDir)

	// base 4 + supporting repo = 5
	jtesting.AssertEqual(t, len(mounts), 5)
	jtesting.AssertEqual(t, mounts[4].Source, supportDir)
	jtesting.AssertEqual(t, mounts[4].Target, "/repos/vicky")
	jtesting.AssertEqual(t, mounts[4].ReadOnly, false)
}

func TestSessionMountsSupportingRepoMissingDir(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}
	cfg = Config{}

	profile := Profile{
		Repos: []string{"git@github.com:zoobzio/flux.git"},
	}
	repoDir := t.TempDir()

	mounts := SessionMounts(profile, "blue", repoDir)

	// base 4 only; missing dir is skipped
	jtesting.AssertEqual(t, len(mounts), 4)
}

func TestSessionEnvFull(t *testing.T) {
	profile := Profile{
		Git: GitConfig{Name: "Rockhopper", Email: "rock@example.com"},
	}
	e := SessionEnv(profile, "blue")

	jtesting.AssertEqual(t, e["JACK_AGENT"], "blue")
	jtesting.AssertEqual(t, e["GIT_AUTHOR_NAME"], "Rockhopper")
	jtesting.AssertEqual(t, e["GIT_COMMITTER_NAME"], "Rockhopper")
	jtesting.AssertEqual(t, e["GIT_AUTHOR_EMAIL"], "rock@example.com")
	jtesting.AssertEqual(t, e["GIT_COMMITTER_EMAIL"], "rock@example.com")
}

func TestSessionEnvEmpty(t *testing.T) {
	profile := Profile{}
	e := SessionEnv(profile, "")

	jtesting.AssertEqual(t, len(e), 0)
}

func TestSessionEnvPartial(t *testing.T) {
	profile := Profile{
		Git: GitConfig{Name: "Rockhopper"},
	}
	e := SessionEnv(profile, "blue")

	jtesting.AssertEqual(t, e["JACK_AGENT"], "blue")
	jtesting.AssertEqual(t, e["GIT_AUTHOR_NAME"], "Rockhopper")
	jtesting.AssertEqual(t, e["GIT_COMMITTER_NAME"], "Rockhopper")
	_, hasEmail := e["GIT_AUTHOR_EMAIL"]
	jtesting.AssertEqual(t, hasEmail, false)
}

func TestDockerExecCmd(t *testing.T) {
	got := DockerExecCmd("jack-blue-vicky", "claude --dangerously-skip-permissions")
	jtesting.AssertEqual(t, got, `docker exec -it jack-blue-vicky sh -c "claude --dangerously-skip-permissions"`)
}

func TestDockerExecCmdQuoting(t *testing.T) {
	got := DockerExecCmd("mycontainer", "echo hello")
	jtesting.AssertEqual(t, strings.Contains(got, `"echo hello"`), true)
	jtesting.AssertEqual(t, strings.HasPrefix(got, "docker exec -it mycontainer sh -c"), true)
}

func TestToolsVolume(t *testing.T) {
	v := ToolsVolume("blue", "vicky")
	jtesting.AssertEqual(t, v.Name, "jack-blue-vicky-tools")
	jtesting.AssertEqual(t, v.Target, "/home/jack/.local/bin")
}
