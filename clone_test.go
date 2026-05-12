//go:build testing

package jack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func noopCloner(_, _ string) error     { return nil }
func noopLinker(_, _ string) error     { return nil }
func noopDescWriter(_, _ string) error { return nil }

func noopRegLoader() (*Registry, error) { return &Registry{}, nil }
func noopRegSaver(_ *Registry) error    { return nil }

// setupAgentFixtures creates the agent config directory needed for applyAgent.
func setupAgentFixtures(t *testing.T) {
	t.Helper()
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}

	for name := range cfg.Profiles {
		agentDir := filepath.Join(configDir, "agents", name)
		_ = os.MkdirAll(agentDir, 0o750)
		_ = os.WriteFile(filepath.Join(agentDir, "CLAUDE.md"), []byte("agent soul"), 0o600)
	}
}

func TestRunCloneUnknownAgent(t *testing.T) {
	newTestConfig()
	setupAgentFixtures(t)
	err := runClone(context.Background(), "git@github.com:zoobzio/vicky.git", []string{"bogus"}, false,
		noopCloner, noopLinker, noopChecker, noopKiller,
		noopDescWriter, noopRegLoader, noopRegSaver, noopImageBuilder)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "unknown agent"), true)
}

func TestRunCloneSuccess(t *testing.T) {
	newTestConfig()
	setupAgentFixtures(t)

	var clonedURLs, clonedDirs []string
	cloner := func(url, dir string) error {
		clonedURLs = append(clonedURLs, url)
		clonedDirs = append(clonedDirs, dir)
		return nil
	}

	var savedReg *Registry
	saver := func(r *Registry) error {
		savedReg = r
		return nil
	}

	err := runClone(context.Background(), "git@github.com:zoobzio/vicky.git", []string{"blue"}, false,
		cloner, noopLinker, noopChecker, noopKiller,
		noopDescWriter, noopRegLoader, saver, noopImageBuilder)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(clonedURLs), 1)
	jtesting.AssertEqual(t, strings.HasSuffix(clonedDirs[0], "blue/vicky"), true)
	jtesting.AssertEqual(t, savedReg != nil, true)
	jtesting.AssertEqual(t, savedReg.Find("blue", "vicky") != nil, true)
}

func TestRunCloneMultipleAgents(t *testing.T) {
	cfg = Config{
		Profiles: map[string]Profile{
			"blue": {Git: GitConfig{Name: "Rockhopper", Email: "rock@example.com"}},
			"red":  {Git: GitConfig{Name: "Mother", Email: "mother@example.com"}},
		},
	}
	setupAgentFixtures(t)

	var savedReg *Registry
	saver := func(r *Registry) error {
		savedReg = r
		return nil
	}

	err := runClone(context.Background(), "git@github.com:zoobzio/vicky.git", []string{"blue", "red"}, false,
		noopCloner, noopLinker, noopChecker, noopKiller,
		noopDescWriter, noopRegLoader, saver, noopImageBuilder)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(savedReg.Projects), 2)
	jtesting.AssertEqual(t, savedReg.Find("blue", "vicky") != nil, true)
	jtesting.AssertEqual(t, savedReg.Find("red", "vicky") != nil, true)
}

func TestRunCloneDescription(t *testing.T) {
	newTestConfig()
	setupAgentFixtures(t)

	var descPath, descContent string
	descWriter := func(path, content string) error {
		descPath = path
		descContent = content
		return nil
	}

	err := runClone(context.Background(), "git@github.com:zoobzio/vicky.git", []string{"blue"}, false,
		noopCloner, noopLinker, noopChecker, noopKiller,
		descWriter, noopRegLoader, noopRegSaver, noopImageBuilder)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, strings.Contains(descPath, ".jack/description.txt"), true)
	jtesting.AssertEqual(t, strings.Contains(descContent, "agent=blue"), true)
}

func TestRunCloneValidationFailsMissingProfile(t *testing.T) {
	newTestConfig()
	env = Env{ConfigDir: t.TempDir(), DataDir: t.TempDir()}

	err := runClone(context.Background(), "git@github.com:zoobzio/vicky.git", []string{"bogus"}, false,
		noopCloner, noopLinker, noopChecker, noopKiller,
		noopDescWriter, noopRegLoader, noopRegSaver, noopImageBuilder)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "unknown agent"), true)
}

func TestRunCloneSkipsExisting(t *testing.T) {
	newTestConfig()
	setupAgentFixtures(t)

	// Pre-create the repo directory to simulate a previous clone.
	dir := filepath.Join(env.dataDir(), "blue", "vicky")
	_ = os.MkdirAll(dir, 0o750)

	var cloned bool
	cloner := func(_, _ string) error {
		cloned = true
		return nil
	}

	err := runClone(context.Background(), "git@github.com:zoobzio/vicky.git", []string{"blue"}, false,
		cloner, noopLinker, noopChecker, noopKiller,
		noopDescWriter, noopRegLoader, noopRegSaver, noopImageBuilder)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, cloned, false)
}

func TestRunCloneForceReplacesExisting(t *testing.T) {
	newTestConfig()
	setupAgentFixtures(t)

	// Pre-create the repo directory to simulate a previous clone.
	dir := filepath.Join(env.dataDir(), "blue", "vicky")
	_ = os.MkdirAll(dir, 0o750)

	var cloned bool
	cloner := func(_, _ string) error {
		cloned = true
		return nil
	}

	var killed bool
	killer := func(_ string) error {
		killed = true
		return nil
	}

	hasSession := func(_ string) bool { return true }

	err := runClone(context.Background(), "git@github.com:zoobzio/vicky.git", []string{"blue"}, true,
		cloner, noopLinker, hasSession, killer,
		noopDescWriter, noopRegLoader, noopRegSaver, noopImageBuilder)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, cloned, true)
	jtesting.AssertEqual(t, killed, true)
}
