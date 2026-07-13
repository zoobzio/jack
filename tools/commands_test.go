package tools

import (
	"strings"
	"testing"

	"github.com/zoobzio/jack/domain"
)

// mainIdentity builds a main-clone identity.
func mainIdentity(t *testing.T) *domain.Identity {
	t.Helper()
	id, err := domain.NewIdentity(domain.Agent("alex"), domain.Repo("jack"))
	if err != nil {
		t.Fatalf("NewIdentity: %v", err)
	}
	return id
}

func TestBootstrap(t *testing.T) {
	got := For(mainIdentity(t)).Bootstrap()

	if len(got) != 3 {
		t.Fatalf("Bootstrap len = %d, want 3 (%q)", len(got), got)
	}
	if got[0] != "sh" || got[1] != "-c" {
		t.Fatalf("Bootstrap prefix = %q, %q; want \"sh\", \"-c\"", got[0], got[1])
	}

	script := got[2]
	for _, want := range []string{
		"step ca bootstrap",
		"step ca certificate",
		"step ca renew --daemon",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("Bootstrap script missing %q\nscript:\n%s", want, script)
		}
	}
}

func TestSetup(t *testing.T) {
	const configDir = "/etc/jack"
	const mount = "/root/.config/jack"

	got := For(mainIdentity(t)).Setup(configDir)

	if len(got) != 3 {
		t.Fatalf("Setup len = %d, want 3", len(got))
	}

	want := []Setup{
		{
			HostPath: configDir + "/setup.sh",
			Command:  []string{"sh", mount + "/setup.sh"},
			Label:    "global setup",
		},
		{
			HostPath: configDir + "/agents/alex/setup.sh",
			Command:  []string{"sh", mount + "/agents/alex/setup.sh"},
			Label:    "agent setup for alex",
		},
		{
			HostPath: configDir + "/projects/jack/dev.sh",
			Command:  []string{"sh", mount + "/projects/jack/dev.sh"},
			Label:    "project setup for jack",
		},
	}

	for i, w := range want {
		g := got[i]
		if g.HostPath != w.HostPath {
			t.Errorf("Setup[%d].HostPath = %q, want %q", i, g.HostPath, w.HostPath)
		}
		if len(g.Command) != len(w.Command) {
			t.Errorf("Setup[%d].Command = %q, want %q", i, g.Command, w.Command)
			continue
		}
		for j := range w.Command {
			if g.Command[j] != w.Command[j] {
				t.Errorf("Setup[%d].Command[%d] = %q, want %q", i, j, g.Command[j], w.Command[j])
			}
		}
		if g.Label != w.Label {
			t.Errorf("Setup[%d].Label = %q, want %q", i, g.Label, w.Label)
		}
	}
}
