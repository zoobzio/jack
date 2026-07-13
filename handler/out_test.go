package handler

import (
	"context"
	"strings"
	"testing"
)

func TestOutSessionNotFound(t *testing.T) {
	tm := &fakeTmux{HasResult: false}
	app := testApp(testEnv(t), nil, nil, tm, nil)

	err := out(context.Background(), app, "", "alex", "jack")
	if err == nil {
		t.Fatal("out with absent session = nil error, want error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention \"not found\"", err)
	}
	if len(tm.KillNames) != 0 {
		t.Errorf("Kill called %v, want none for an absent session", tm.KillNames)
	}
}

func TestOutMainSessionStopsContainer(t *testing.T) {
	tm := &fakeTmux{HasResult: true}
	d := &fakeDocker{}
	app := testApp(testEnv(t), nil, d, tm, nil)

	if err := out(context.Background(), app, "", "alex", "jack"); err != nil {
		t.Fatalf("out returned error: %v", err)
	}

	// Session name for a main session is "<agent>-<repo>".
	if len(tm.KillNames) != 1 || tm.KillNames[0] != "alex-jack" {
		t.Errorf("Kill = %v, want [alex-jack]", tm.KillNames)
	}
	// A main session stops its container jack-<agent>-<repo>.
	if len(d.StopNames) != 1 || d.StopNames[0] != "jack-alex-jack" {
		t.Errorf("Stop = %v, want [jack-alex-jack]", d.StopNames)
	}
}

func TestOutPositionalNameKills(t *testing.T) {
	tm := &fakeTmux{HasResult: true}
	d := &fakeDocker{}
	app := testApp(testEnv(t), nil, d, tm, nil)

	// Positional name, no agent/project flags.
	if err := out(context.Background(), app, "alex-jack", "", ""); err != nil {
		t.Fatalf("out returned error: %v", err)
	}

	if len(tm.KillNames) != 1 || tm.KillNames[0] != "alex-jack" {
		t.Errorf("Kill = %v, want [alex-jack]", tm.KillNames)
	}
	// The container name is recovered from the positional session name.
	if len(d.StopNames) != 1 || d.StopNames[0] != "jack-alex-jack" {
		t.Errorf("Stop = %v, want [jack-alex-jack]", d.StopNames)
	}
}

func TestOutRequiresName(t *testing.T) {
	app := testApp(testEnv(t), nil, nil, &fakeTmux{}, nil)

	err := out(context.Background(), app, "", "", "")
	if err == nil {
		t.Fatal("out with no name and no flags = nil error, want error")
	}
}
