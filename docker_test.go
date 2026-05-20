//go:build testing

package jack

import (
	"os"
	"path/filepath"
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

	mounts := SessionMounts(profile, "blue", "vicky", repoDir)

	home, _ := os.UserHomeDir()
	// .claude, .claude.json, agent .claude/, repo = 4
	jtesting.AssertEqual(t, len(mounts), 4)
	jtesting.AssertEqual(t, mounts[0].Source, filepath.Join(home, ".claude"))
	jtesting.AssertEqual(t, mounts[0].Target, "/root/.claude")
	jtesting.AssertEqual(t, mounts[1].Source, filepath.Join(home, ".claude.json"))
	jtesting.AssertEqual(t, mounts[1].Target, "/root/.claude.json")
	jtesting.AssertEqual(t, mounts[2].Target, "/root/workspace/.claude")
	jtesting.AssertEqual(t, mounts[2].ReadOnly, true)
	jtesting.AssertEqual(t, mounts[3].Source, repoDir)
	jtesting.AssertEqual(t, mounts[3].Target, "/root/workspace/vicky")
	jtesting.AssertEqual(t, mounts[3].ReadOnly, false)
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

	mounts := SessionMounts(profile, "blue", "vicky", repoDir)

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

	mounts := SessionMounts(profile, "blue", "vicky", repoDir)

	// base 4 only; missing dir is skipped
	jtesting.AssertEqual(t, len(mounts), 4)
}

func TestSessionEnvFull(t *testing.T) {
	profile := Profile{
		Git: GitConfig{Name: "Rockhopper", Email: "rock@example.com"},
	}
	e := SessionEnv(profile, "blue", CAConfig{})

	jtesting.AssertEqual(t, e["JACK_AGENT"], "blue")
	jtesting.AssertEqual(t, e["GIT_AUTHOR_NAME"], "Rockhopper")
	jtesting.AssertEqual(t, e["GIT_COMMITTER_NAME"], "Rockhopper")
	jtesting.AssertEqual(t, e["GIT_AUTHOR_EMAIL"], "rock@example.com")
	jtesting.AssertEqual(t, e["GIT_COMMITTER_EMAIL"], "rock@example.com")
}

func TestSessionEnvEmpty(t *testing.T) {
	profile := Profile{}
	e := SessionEnv(profile, "", CAConfig{})

	jtesting.AssertEqual(t, len(e), 0)
}

func TestSessionEnvPartial(t *testing.T) {
	profile := Profile{
		Git: GitConfig{Name: "Rockhopper"},
	}
	e := SessionEnv(profile, "blue", CAConfig{})

	jtesting.AssertEqual(t, e["JACK_AGENT"], "blue")
	jtesting.AssertEqual(t, e["GIT_AUTHOR_NAME"], "Rockhopper")
	jtesting.AssertEqual(t, e["GIT_COMMITTER_NAME"], "Rockhopper")
	_, hasEmail := e["GIT_AUTHOR_EMAIL"]
	jtesting.AssertEqual(t, hasEmail, false)
}

func TestSessionEnvWithCA(t *testing.T) {
	profile := Profile{}
	ca := CAConfig{
		URL:         "https://ca.example.com",
		Fingerprint: "abc123",
		Provisioner: "jack",
	}
	e := SessionEnv(profile, "blue", ca)

	jtesting.AssertEqual(t, e["JACK_CA_URL"], "https://ca.example.com")
	jtesting.AssertEqual(t, e["JACK_CA_FINGERPRINT"], "abc123")
	jtesting.AssertEqual(t, e["JACK_CA_PROVISIONER"], "jack")
}

func TestDockerExecCmd(t *testing.T) {
	got := DockerExecCmd("jack-blue-vicky", "/workspace", "claude")
	jtesting.AssertEqual(t, got, "docker exec -it -w /workspace jack-blue-vicky claude")
}

func TestDockerExecCmdNoArgs(t *testing.T) {
	got := DockerExecCmd("mycontainer", "/root", "bash")
	jtesting.AssertEqual(t, got, "docker exec -it -w /root mycontainer bash")
}

func TestToolsVolume(t *testing.T) {
	v := ToolsVolume("blue", "vicky")
	jtesting.AssertEqual(t, v.Name, "jack-blue-vicky-tools")
	jtesting.AssertEqual(t, v.Target, "/root/.jack/bin")
}
