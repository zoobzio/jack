// Package tools builds the shell commands jack runs inside a session's
// container — certificate bootstrap and setup scripts. It only constructs
// commands; executing them (via the Docker boundary) and deciding when to run
// them is the caller's job, which keeps this package free of I/O and easy to
// test.
package tools

import (
	"path/filepath"

	"github.com/zoobzio/jack/domain"
)

// Setup is one setup script to run on fresh container start: the host file whose
// presence gates it, the command that runs it inside the container, and a label
// for progress output.
type Setup struct {
	HostPath string
	Label    string
	Command  []string
}

// Commands builds the container-side commands for a single session identity.
type Commands struct {
	id *domain.Identity
}

// For binds the commands to a session identity.
func For(id *domain.Identity) Commands {
	return Commands{id: id}
}

// Bootstrap returns the command that bootstraps the agent certificate and starts
// its renew daemon. It reads the JACK_CA_* and JACK_AGENT env vars injected into
// the container by the Spec, so it takes no arguments.
func (c Commands) Bootstrap() []string {
	const sh = `set -e
step ca bootstrap --ca-url "$JACK_CA_URL" --fingerprint "$JACK_CA_FINGERPRINT" --force
step ca certificate "$JACK_AGENT" /root/.jack/certs/cert.pem /root/.jack/certs/key.pem \
  --provisioner "$JACK_CA_PROVISIONER" --force
step ca renew --daemon \
  --cert /root/.jack/certs/cert.pem \
  --key /root/.jack/certs/key.pem &
`
	return []string{"sh", "-c", sh}
}

// Setup returns the ordered setup scripts for the session — global, then agent,
// then project — sourced from configDir on the host and run from the read-only
// config mount inside the container. The caller runs each whose HostPath exists.
func (c Commands) Setup(configDir string) []Setup {
	const mount = domain.ContainerHome + "/.config/jack"
	agent, repo := string(c.id.Agent()), string(c.id.Repo())
	return []Setup{
		{
			HostPath: filepath.Join(configDir, "setup.sh"),
			Command:  []string{"sh", mount + "/setup.sh"},
			Label:    "global setup",
		},
		{
			HostPath: filepath.Join(configDir, "agents", agent, "setup.sh"),
			Command:  []string{"sh", mount + "/agents/" + agent + "/setup.sh"},
			Label:    "agent setup for " + agent,
		},
		{
			HostPath: filepath.Join(configDir, "projects", repo, "dev.sh"),
			Command:  []string{"sh", mount + "/projects/" + repo + "/dev.sh"},
			Label:    "project setup for " + repo,
		},
	}
}
