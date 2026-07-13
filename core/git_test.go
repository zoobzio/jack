package core

import "testing"

func TestNewGitNonNil(t *testing.T) {
	if NewGit() == nil {
		t.Fatal("NewGit returned nil")
	}
}
