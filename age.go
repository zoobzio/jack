package jack

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TokenEncrypter encrypts a plaintext token using an SSH public key and writes
// the ciphertext to the given path.
type TokenEncrypter func(token, pubKeyPath, outPath string) error

// TokenDecrypter decrypts an age-encrypted file using an SSH private key and
// returns the plaintext token.
type TokenDecrypter func(privKeyPath, agePath string) (string, error)

// DescriptionWriter writes a session description to a file.
type DescriptionWriter func(path, content string) error

func ageEncrypt(token, pubKeyPath, outPath string) error {
	dir := filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	cmd := exec.CommandContext(context.Background(), "age", "-R", pubKeyPath, "-o", outPath)
	cmd.Stdin = strings.NewReader(token)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ageDecrypt(privKeyPath, agePath string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "age", "-d", "-i", privKeyPath, agePath)
	var stdout bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("age decrypt: %w", err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func writeDescription(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func tokenAgePath(repoDir string) string {
	return filepath.Join(repoDir, ".jack", "token.age")
}

func ghTokenAgePath(teamName string) string {
	return filepath.Join(env.configDir(), "teams", teamName, ".github-token.age")
}

func descriptionPath(repoDir string) string {
	return filepath.Join(repoDir, ".jack", "description.txt")
}
