//go:build testing

package jack

import (
	"os"
	"path/filepath"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestDescriptionPath(t *testing.T) {
	got := descriptionPath("/some/repo")
	want := filepath.Join("/some/repo", ".jack", "description.txt")
	jtesting.AssertEqual(t, got, want)
}

func TestWriteDescription(t *testing.T) {
	t.Run("creates dirs and writes content", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, ".jack", "description.txt")

		err := writeDescription(path, "agent=blue repo=vicky")
		jtesting.AssertNoError(t, err)

		data, err := os.ReadFile(path)
		jtesting.AssertNoError(t, err)
		jtesting.AssertEqual(t, string(data), "agent=blue repo=vicky")
	})

	t.Run("overwrites existing description", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, ".jack", "description.txt")

		jtesting.AssertNoError(t, writeDescription(path, "first"))
		jtesting.AssertNoError(t, writeDescription(path, "second"))

		data, err := os.ReadFile(path)
		jtesting.AssertNoError(t, err)
		jtesting.AssertEqual(t, string(data), "second")
	})
}
