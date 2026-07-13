package core

import "testing"

func TestNewDockerConfigured(t *testing.T) {
	d := NewDocker()
	if d == nil {
		t.Fatal("NewDocker returned nil")
	}
	if d.image == "" {
		t.Error("docker image is empty")
	}
	if d.image != image {
		t.Errorf("docker image = %q, want %q", d.image, image)
	}
	if d.dockerfile == "" {
		t.Error("docker dockerfile is empty")
	}
	if d.dockerfile != dockerfile {
		t.Error("docker dockerfile does not match the package dockerfile constant")
	}
}

func TestNewDockerSatisfiesInterface(t *testing.T) {
	var _ Docker = NewDocker()
}
