//go:build testing

package jack

import (
	"context"
)

// Shared test helpers used across multiple test files.

func newTestConfig() {
	cfg = Config{
		Profiles: map[string]Profile{
			"blue": {
				Git: GitConfig{Name: "Rockhopper", Email: "rock@example.com"},
			},
		},
	}
}

func noopChecker(string) bool                          { return false }
func existsChecker(string) bool                        { return true }
func noopCreator(_, _, _ string) error                  { return nil }
func noopAttacher(_ string) error                      { return nil }
func noopKiller(_ string) error                        { return nil }
func noopContainerRunner(_ string, _ []Mount, _ []Volume, _ map[string]string) error { return nil }
func noopContainerExecer(_ string, _ []string) error   { return nil }
func noopContainerStopper(_ string) error              { return nil }
func noopContainerChecker(_ string) (bool, bool)       { return false, false }
func noopImageBuilder(_ context.Context) error         { return nil }
