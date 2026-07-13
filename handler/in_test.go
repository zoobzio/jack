package handler

import (
	"context"
	"testing"
)

func TestInSessionExistsAttaches(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	env := testEnv(t)

	tm := &fakeTmux{HasResult: true}
	d := &fakeDocker{}
	app := testApp(env, profileConfig("alex"), d, tm, &fakeGit{})

	if err := in(context.Background(), app, "alex", "jack"); err != nil {
		t.Fatalf("in returned error: %v", err)
	}

	if len(tm.AttachNames) != 1 || tm.AttachNames[0] != "alex-jack" {
		t.Errorf("Attach = %v, want [alex-jack]", tm.AttachNames)
	}
	if len(d.RunSpecs) != 0 {
		t.Errorf("Run called %d times, want 0 for an existing session", len(d.RunSpecs))
	}
	if len(tm.CreateCalls) != 0 {
		t.Errorf("Create called for an existing session: %v", tm.CreateCalls)
	}
}

func TestInStartsContainerAndCreatesSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	env := testEnv(t)

	// Session absent and container not running.
	tm := &fakeTmux{HasResult: false}
	d := &fakeDocker{RunningResult: false}
	app := testApp(env, profileConfig("alex"), d, tm, &fakeGit{})

	if err := in(context.Background(), app, "alex", "jack"); err != nil {
		t.Fatalf("in returned error: %v", err)
	}

	if len(d.RunSpecs) != 1 {
		t.Fatalf("Run called %d times, want 1", len(d.RunSpecs))
	}
	if d.RunSpecs[0].Name != "jack-alex-jack" {
		t.Errorf("Run spec name = %q, want jack-alex-jack", d.RunSpecs[0].Name)
	}
	if len(tm.CreateCalls) != 1 || tm.CreateCalls[0].Name != "alex-jack" {
		t.Errorf("Create = %v, want one call for alex-jack", tm.CreateCalls)
	}
	if len(tm.AttachNames) != 1 || tm.AttachNames[0] != "alex-jack" {
		t.Errorf("Attach = %v, want [alex-jack]", tm.AttachNames)
	}
	// CA.URL is empty and no setup scripts exist, so nothing was exec'd.
	if len(d.ExecCalls) != 0 {
		t.Errorf("Exec called %d times, want 0", len(d.ExecCalls))
	}
}
