package main

import (
	"testing"

	"github.com/zoobzio/jack/core"
	"github.com/zoobzio/jack/handler"
)

// TestWiring exercises the same handler registration that main() performs,
// without the os.Exit / Execute path. It asserts that each handler mounts its
// subcommand onto the app's root command.
func TestWiring(t *testing.T) {
	app, err := core.NewApp()
	if err != nil {
		t.Fatalf("core.NewApp: %v", err)
	}

	handler.Init(app)
	handler.Clone(app)
	handler.Status(app)
	handler.In(app)
	handler.Out(app)
	handler.Refresh(app)
	handler.Kill(app)

	got := make(map[string]bool)
	for _, cmd := range app.Root().Commands() {
		got[cmd.Name()] = true
	}

	for _, want := range []string{"init", "clone", "status", "in", "out", "refresh", "kill"} {
		if !got[want] {
			t.Errorf("root command missing subcommand %q; have %v", want, got)
		}
	}
}
