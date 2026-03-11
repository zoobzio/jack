//go:build testing

package jack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoobzio/jack/msg"
	jtesting "github.com/zoobzio/jack/testing"
)

func TestRepoName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"scp with .git", "git@github.com:zoobzio/vicky.git", "vicky"},
		{"scp without .git", "git@github.com:zoobzio/vicky", "vicky"},
		{"https with .git", "https://github.com/zoobzio/vicky.git", "vicky"},
		{"https without .git", "https://github.com/zoobzio/vicky", "vicky"},
		{"ssh protocol", "ssh://git@github.com/zoobzio/vicky.git", "vicky"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jtesting.AssertEqual(t, repoName(tt.input), tt.want)
		})
	}
}

func noopCloner(_, _ string) error      { return nil }
func noopCopier(_, _ string) error       { return nil }
func noopEncrypter(_, _, _ string) error { return nil }
func noopDescWriter(_, _ string) error   { return nil }

func noopRegisterer(_, _, _ string) (*msg.Registration, error) {
	return &msg.Registration{AccessToken: "tok_test"}, nil
}

// setupGovernanceFixtures creates the governance, projects, skill, and team
// directories needed for clone validation and team application to pass.
func setupGovernanceFixtures(t *testing.T, repo string, skills []string) {
	t.Helper()
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}

	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("x"), 0o600)
	_ = os.MkdirAll(filepath.Join(configDir, "projects", repo), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "projects", repo, "MISSION.md"), []byte("x"), 0o600)

	// Team skills directories with skill dirs and ORDERS.md.
	for name := range cfg.Profiles {
		teamSkillsDir := filepath.Join(configDir, "teams", name, "skills")
		_ = os.MkdirAll(teamSkillsDir, 0o750)
		_ = os.WriteFile(filepath.Join(configDir, "teams", name, "ORDERS.md"), []byte("x"), 0o600)
		for _, skill := range skills {
			skillDir := filepath.Join(teamSkillsDir, skill)
			_ = os.MkdirAll(skillDir, 0o750)
			_ = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("x"), 0o600)
		}
		_ = os.MkdirAll(filepath.Join(configDir, "teams", name, "agents"), 0o750)
	}
}

func TestRunCloneUnknownTeam(t *testing.T) {
	newTestConfig()
	setupGovernanceFixtures(t, "vicky", []string{"commit", "pr"})
	err := runClone("git@github.com:zoobzio/vicky.git", []string{"bogus"}, noopCloner, noopCopier, noopChecker, noopCreator, noopAdder, noopRegisterer, noopEncrypter, noopDescWriter, noopDecrypter)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "team skills directory not found"), true)
}

func TestRunCloneSuccess(t *testing.T) {
	newTestConfig()
	setupGovernanceFixtures(t, "vicky", []string{"commit", "pr"})

	var clonedURLs, clonedDirs []string
	var createdSessions []string

	cloner := func(url, dir string) error {
		clonedURLs = append(clonedURLs, url)
		clonedDirs = append(clonedDirs, dir)
		return nil
	}
	creator := func(name, dir, shellCmd string) error {
		createdSessions = append(createdSessions, name)
		return nil
	}

	err := runClone("git@github.com:zoobzio/vicky.git", []string{"blue"}, cloner, noopCopier, noopChecker, creator, noopAdder, noopRegisterer, noopEncrypter, noopDescWriter, noopDecrypter)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(clonedURLs), 1)
	jtesting.AssertEqual(t, clonedURLs[0], "git@github.com:zoobzio/vicky.git")
	jtesting.AssertEqual(t, strings.HasSuffix(clonedDirs[0], "blue/vicky"), true)
	jtesting.AssertEqual(t, len(createdSessions), 1)
	jtesting.AssertEqual(t, createdSessions[0], "blue-vicky")
}

func TestRunCloneMultipleTeams(t *testing.T) {
	cfg = Config{
		Profiles: map[string]Profile{
			"blue": {Git: GitConfig{Name: "Rockhopper", Email: "rock@example.com"}},
			"red":  {Git: GitConfig{Name: "Mother", Email: "mother@example.com"}},
		},
	}
	setupGovernanceFixtures(t, "vicky", []string{"commit"})

	var createdSessions []string
	creator := func(name, dir, shellCmd string) error {
		createdSessions = append(createdSessions, name)
		return nil
	}

	err := runClone("git@github.com:zoobzio/vicky.git", []string{"blue", "red"}, noopCloner, noopCopier, noopChecker, creator, noopAdder, noopRegisterer, noopEncrypter, noopDescWriter, noopDecrypter)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(createdSessions), 2)
	jtesting.AssertEqual(t, createdSessions[0], "blue-vicky")
	jtesting.AssertEqual(t, createdSessions[1], "red-vicky")
}

func TestRunCloneRegistersAndEncrypts(t *testing.T) {
	newTestConfig()
	setupGovernanceFixtures(t, "vicky", []string{"commit", "pr"})

	var registeredUsername string
	registerer := func(user, pass, token string) (*msg.Registration, error) {
		registeredUsername = user
		return &msg.Registration{AccessToken: "tok_new"}, nil
	}

	var encryptedToken, encryptedPubKey, encryptedPath string
	encrypter := func(token, pubKey, outPath string) error {
		encryptedToken = token
		encryptedPubKey = pubKey
		encryptedPath = outPath
		return nil
	}

	var descPath, descContent string
	descWriter := func(path, content string) error {
		descPath = path
		descContent = content
		return nil
	}

	err := runClone("git@github.com:zoobzio/vicky.git", []string{"blue"},
		noopCloner, noopCopier, noopChecker, noopCreator, noopAdder,
		registerer, encrypter, descWriter, noopDecrypter)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, registeredUsername, "blue-vicky")
	jtesting.AssertEqual(t, encryptedToken, "tok_new")
	jtesting.AssertEqual(t, strings.HasSuffix(encryptedPubKey, "id_rock.pub"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(encryptedPath, ".jack/token.age"), true)
	jtesting.AssertEqual(t, strings.Contains(descPath, ".jack/description.txt"), true)
	jtesting.AssertEqual(t, strings.Contains(descContent, "team=blue"), true)
}

func TestRunCloneValidationFailsMissingGovernance(t *testing.T) {
	newTestConfig()
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: t.TempDir()}

	err := runClone("git@github.com:zoobzio/vicky.git", []string{"blue"},
		noopCloner, noopCopier, noopChecker, noopCreator, noopAdder,
		noopRegisterer, noopEncrypter, noopDescWriter, noopDecrypter)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "governance"), true)
}
