//go:build testing

package jack

import (
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

func TestRunCloneUnknownTeam(t *testing.T) {
	newTestConfig()
	err := runClone("git@github.com:zoobzio/vicky.git", []string{"bogus"}, "developer", noopCloner, noopCopier, noopChecker, noopCreator, noopAdder)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "unknown team"), true)
}

func TestRunCloneUnknownRole(t *testing.T) {
	newTestConfig()
	err := runClone("git@github.com:zoobzio/vicky.git", []string{"blue"}, "bogus", noopCloner, noopCopier, noopChecker, noopCreator, noopAdder)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "unknown role"), true)
}

func TestRunCloneSuccess(t *testing.T) {
	newTestConfig()

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

	err := runClone("git@github.com:zoobzio/vicky.git", []string{"blue"}, "developer", cloner, noopCopier, noopChecker, creator, noopAdder)
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
			"rockhopper": {Git: GitConfig{Name: "Rockhopper", Email: "rock@example.com"}},
			"mother":     {Git: GitConfig{Name: "Mother", Email: "mother@example.com"}},
		},
		Roles: map[string]Role{
			"developer": {Skills: []string{"commit"}},
		},
		Teams: map[string]Team{
			"blue": {Profile: "rockhopper"},
			"red":  {Profile: "mother"},
		},
	}

	var createdSessions []string
	creator := func(name, dir, shellCmd string) error {
		createdSessions = append(createdSessions, name)
		return nil
	}

	err := runClone("git@github.com:zoobzio/vicky.git", []string{"blue", "red"}, "developer", noopCloner, noopCopier, noopChecker, creator, noopAdder)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(createdSessions), 2)
	jtesting.AssertEqual(t, createdSessions[0], "blue-vicky")
	jtesting.AssertEqual(t, createdSessions[1], "red-vicky")
}

func noopCloner(_, _ string) error { return nil }
func noopCopier(_, _ string) error { return nil }
