package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/zoobzio/jack/config"
)

func TestKillForceRemovesEverything(t *testing.T) {
	env := testEnv(t)
	d := &fakeDocker{}
	tm := &fakeTmux{HasResult: true}
	app := testApp(env, nil, d, tm, nil)

	// Seed the clone on disk and a registry entry to be torn down.
	dir := filepath.Join(env.DataDir, "alex", "jack")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	reg, err := config.NewRegistry(env.RegistryPath)
	if err != nil {
		t.Fatal(err)
	}
	reg.Add("alex", "jack", "https://host/u/jack.git")
	if err := reg.Save(); err != nil {
		t.Fatal(err)
	}

	if err := kill(context.Background(), app, "alex", "jack", true); err != nil {
		t.Fatalf("kill returned error: %v", err)
	}

	if len(tm.KillNames) != 1 || tm.KillNames[0] != "alex-jack" {
		t.Errorf("Kill = %v, want [alex-jack]", tm.KillNames)
	}
	if len(d.StopNames) != 1 || d.StopNames[0] != "jack-alex-jack" {
		t.Errorf("Stop = %v, want [jack-alex-jack]", d.StopNames)
	}
	if len(d.RemoveVolumeNames) != 1 || d.RemoveVolumeNames[0] != "jack-alex-jack-tools" {
		t.Errorf("RemoveVolume = %v, want [jack-alex-jack-tools]", d.RemoveVolumeNames)
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Errorf("clone dir still exists (%v), want removed", statErr)
	}
	reg2, err := config.NewRegistry(env.RegistryPath)
	if err != nil {
		t.Fatal(err)
	}
	if reg2.Find("alex", "jack") != nil {
		t.Errorf("registry still has alex/jack, want removed")
	}
}

func TestKillForceSkipsKillForAbsentSession(t *testing.T) {
	env := testEnv(t)
	d := &fakeDocker{}
	tm := &fakeTmux{HasResult: false}
	app := testApp(env, nil, d, tm, nil)

	if err := kill(context.Background(), app, "alex", "jack", true); err != nil {
		t.Fatalf("kill returned error: %v", err)
	}

	if len(tm.KillNames) != 0 {
		t.Errorf("Kill called %v, want none for an absent session", tm.KillNames)
	}
	// The container and its volume are still removed even with no session.
	if len(d.StopNames) != 1 {
		t.Errorf("Stop calls = %v, want 1", d.StopNames)
	}
	if len(d.RemoveVolumeNames) != 1 {
		t.Errorf("RemoveVolume calls = %v, want 1", d.RemoveVolumeNames)
	}
}
