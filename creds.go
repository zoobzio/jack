package jack

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const keychainService = "Claude Code-credentials"

// keychainReader reads credentials from a system keychain.
type keychainReader func() ([]byte, error)

// credWriter writes credentials to a path.
type credWriter func(path string, data []byte) error

// readKeychain reads Claude OAuth credentials from the macOS keychain.
func readKeychain() ([]byte, error) {
	cmd := exec.CommandContext(context.Background(), "security", "find-generic-password", "-s", keychainService, "-w")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("reading keychain: %w: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// writeCredFile writes credentials to disk.
func writeCredFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

// syncClaudeCredentials reads Claude OAuth credentials from the macOS
// keychain and writes them to ~/.claude/.credentials.json so that
// containers (which lack keychain access) can authenticate.
// On non-macOS platforms this is a no-op.
func syncClaudeCredentials() error {
	return doSyncCredentials(runtime.GOOS, readKeychain, writeCredFile)
}

// doSyncCredentials is the testable core of syncClaudeCredentials.
func doSyncCredentials(goos string, read keychainReader, write credWriter) error {
	if goos != "darwin" {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	credPath := filepath.Join(home, ".claude", ".credentials.json")

	raw, err := read()
	if err != nil {
		return err
	}

	creds := bytes.TrimSpace(raw)
	if len(creds) == 0 {
		return fmt.Errorf("empty credentials in keychain")
	}

	if err := write(credPath, creds); err != nil {
		return fmt.Errorf("writing credentials: %w", err)
	}

	return nil
}
