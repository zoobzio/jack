package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Repo is a repository name — the short identifier extracted from a git URL and
// used alongside an Agent in container and session names. It is a
// distinct type so that its validation rule travels with it.
type Repo string

// NewRepo extracts the repository name from a git URL, handling both SCP-style
// (git@host:user/repo.git) and standard URLs, and validates the result.
func NewRepo(url string) (Repo, error) {
	name := strings.TrimSuffix(url, ".git")
	if i := strings.LastIndex(name, ":"); i != -1 && !strings.Contains(name, "://") {
		name = name[i+1:]
	}
	if i := strings.LastIndex(name, "/"); i != -1 {
		name = name[i+1:]
	}
	r := Repo(name)
	if err := r.Validate(); err != nil {
		return "", fmt.Errorf("cannot extract repo name from %q: %w", url, err)
	}
	return r, nil
}

// Validate reports whether the repo name is usable. A repo must be non-empty;
// unlike an Agent it may contain '-', since a session name splits on the first
// hyphen (the agent) and keeps the remainder as the repo.
func (r Repo) Validate() error {
	if r == "" {
		return errors.New("a repo must not be an empty string")
	}
	return nil
}
