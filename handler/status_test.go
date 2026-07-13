package handler

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zoobzio/jack/config"
	"github.com/zoobzio/jack/domain"
)

func TestStatusEmptyRegistry(t *testing.T) {
	// No registry file written → empty registry.
	app := testApp(testEnv(t), nil, nil, &fakeTmux{}, nil)

	var buf bytes.Buffer
	if err := status(context.Background(), app, &buf); err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "no projects cloned") {
		t.Errorf("output = %q, want it to contain \"no projects cloned\"", buf.String())
	}
}

func TestStatusRendersAgentTable(t *testing.T) {
	env := testEnv(t)

	// Seed the registry with one agent/repo.
	reg, err := config.NewRegistry(env.RegistryPath)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	reg.Add("alex", "jack", "https://github.com/zoobzio/jack")
	if err := reg.Save(); err != nil {
		t.Fatalf("registry Save: %v", err)
	}

	tm := &fakeTmux{ListResult: []domain.Session{
		{Name: "alex-jack", Activity: time.Now(), Windows: 1},
	}}
	app := testApp(env, nil, nil, tm, nil)

	var buf bytes.Buffer
	if err := status(context.Background(), app, &buf); err != nil {
		t.Fatalf("status returned error: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"alex", "jack", "PROJECT", "SESSION", "STATUS", "CONTAINER"} {
		if !strings.Contains(out, want) {
			t.Errorf("status output missing %q\noutput:\n%s", want, out)
		}
	}
}
