//go:build testing

package jack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func newTestConfig() {
	cfg = Config{
		Profiles: map[string]Profile{
			"blue": {
				Git: GitConfig{Name: "Rockhopper", Email: "rock@example.com"},
				SSH: SSHConfig{Key: "~/.ssh/id_rock"},
			},
		},
	}
}

func noopChecker(string) bool                  { return false }
func existsChecker(string) bool                { return true }
func noopCreator(_, _, _ string) error          { return nil }
func noopAdder(_ string) error                  { return nil }
func noopDecrypter(_, _ string) (string, error) { return "", nil }
func noopAttacher(_ string) error               { return nil }

func TestRunRunUnknownTeam(t *testing.T) {
	newTestConfig()
	err := runRun("vicky", "bogus", "/tmp", true, noopChecker, noopCreator, noopAttacher, noopAdder, noopDecrypter)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "unknown team"), true)
}

func TestRunRunSessionExists(t *testing.T) {
	newTestConfig()
	err := runRun("vicky", "blue", "/tmp", true, existsChecker, noopCreator, noopAttacher, noopAdder, noopDecrypter)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "already exists"), true)
}

func TestRunRunDetached(t *testing.T) {
	newTestConfig()

	var createdName, createdDir, createdCmd string
	var addedKey string

	creator := func(name, dir, shellCmd string) error {
		createdName = name
		createdDir = dir
		createdCmd = shellCmd
		return nil
	}
	adder := func(key string) error {
		addedKey = key
		return nil
	}

	attachCalled := false
	attacher := func(_ string) error {
		attachCalled = true
		return nil
	}

	err := runRun("vicky", "blue", "/home/user/vicky", true, noopChecker, creator, attacher, adder, noopDecrypter)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, createdName, "blue-vicky")
	jtesting.AssertEqual(t, createdDir, "/home/user/vicky")
	jtesting.AssertEqual(t, strings.Contains(addedKey, "id_rock"), true)
	jtesting.AssertEqual(t, strings.Contains(createdCmd, "claude --dangerously-skip-permissions"), true)
	jtesting.AssertEqual(t, attachCalled, false)
}

func TestRunRunAttaches(t *testing.T) {
	newTestConfig()

	var attachedName string
	attacher := func(name string) error {
		attachedName = name
		return nil
	}

	err := runRun("vicky", "blue", "/home/user/vicky", false, noopChecker, noopCreator, attacher, noopAdder, noopDecrypter)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, attachedName, "blue-vicky")
}

func TestRunRunNoSSHKey(t *testing.T) {
	cfg = Config{
		Profiles: map[string]Profile{
			"red": {Git: GitConfig{Name: "Bare", Email: "bare@example.com"}},
		},
	}

	adderCalled := false
	adder := func(_ string) error {
		adderCalled = true
		return nil
	}

	err := runRun("flux", "red", "/tmp", true, noopChecker, noopCreator, noopAttacher, adder, noopDecrypter)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, adderCalled, false)
}

func TestRunRunDecryptsToken(t *testing.T) {
	newTestConfig()

	dir := t.TempDir()
	jackDir := filepath.Join(dir, ".jack")
	_ = os.MkdirAll(jackDir, 0o750)
	_ = os.WriteFile(filepath.Join(jackDir, "token.age"), []byte("encrypted"), 0o600)

	var capturedCmd string
	creator := func(_, _, shellCmd string) error {
		capturedCmd = shellCmd
		return nil
	}
	decrypter := func(privKey, agePath string) (string, error) {
		jtesting.AssertEqual(t, strings.HasSuffix(agePath, "token.age"), true)
		return "tok_decrypted", nil
	}

	err := runRun("testrepo", "blue", dir, true, noopChecker, creator, noopAttacher, noopAdder, decrypter)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, strings.Contains(capturedCmd, "JACK_MSG_TOKEN tok_decrypted"), true)
}

func TestBuildShellCmd(t *testing.T) {
	profile := Profile{
		Git: GitConfig{Name: "Test User", Email: "test@example.com"},
	}

	cmd := buildShellCmd(profile, "/home/user/project", "")

	jtesting.AssertEqual(t, strings.Contains(cmd, `git config user.name "Test User"`), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, `git config user.email "test@example.com"`), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "--ro-bind / /"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "--bind /home/user/project /home/user/project"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "--dev /dev"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "--proc /proc"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "--tmpfs /tmp"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "-- claude --dangerously-skip-permissions"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, " && "), true)
}

func TestBuildShellCmdNoGitConfig(t *testing.T) {
	profile := Profile{}

	cmd := buildShellCmd(profile, "/tmp", "")

	jtesting.AssertEqual(t, strings.Contains(cmd, "git config"), false)
	jtesting.AssertEqual(t, strings.HasPrefix(cmd, "exec bwrap"), true)
}

func TestBuildShellCmdWithToken(t *testing.T) {
	profile := Profile{
		Git: GitConfig{Name: "Test User", Email: "test@example.com"},
	}

	cmd := buildShellCmd(profile, "/home/user/project", "tok_session")
	jtesting.AssertEqual(t, strings.Contains(cmd, "--setenv JACK_MSG_TOKEN tok_session"), true)
}
