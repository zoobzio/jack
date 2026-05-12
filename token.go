package jack

import (
	"os"
	"path/filepath"
)

// DescriptionWriter writes a session description to a file.
type DescriptionWriter func(path, content string) error

func descriptionPath(repoDir string) string {
	return filepath.Join(repoDir, ".jack", "description.txt")
}

func writeDescription(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}
