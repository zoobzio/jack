//go:build testing

package jack

import (
	"os"
	"path/filepath"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestValidateNoProfiles(t *testing.T) {
	c := Config{}
	err := c.Validate()
	jtesting.AssertError(t, err)
}

func TestValidateProfileNameWithHyphen(t *testing.T) {
	c := Config{
		Profiles: map[string]Profile{
			"bad-name": {Git: GitConfig{Name: "Test", Email: "test@example.com"}},
		},
	}
	err := c.Validate()
	jtesting.AssertError(t, err)
}

func TestValidateSuccess(t *testing.T) {
	c := Config{
		Profiles: map[string]Profile{
			"blue": {Git: GitConfig{Name: "Rockhopper", Email: "rock@example.com"}},
		},
	}
	err := c.Validate()
	jtesting.AssertNoError(t, err)
}

func TestValidateMultipleProfilesOneWithHyphen(t *testing.T) {
	c := Config{
		Profiles: map[string]Profile{
			"blue":    {Git: GitConfig{Name: "Rockhopper", Email: "rock@example.com"}},
			"bad-one": {Git: GitConfig{Name: "Bad", Email: "bad@example.com"}},
		},
	}
	err := c.Validate()
	jtesting.AssertError(t, err)
}

func TestInitConfigLoadsValidYAML(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	content := `
profiles:
  blue:
    git:
      name: Rockhopper
      email: rock@example.com
ca:
  url: https://ca.example.com
  provisioner: myprovisioner
`
	err := os.WriteFile(configFile, []byte(content), 0o600)
	jtesting.AssertNoError(t, err)

	cfg = Config{}
	err = initConfig(configFile)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(cfg.Profiles), 1)
	jtesting.AssertEqual(t, cfg.Profiles["blue"].Git.Name, "Rockhopper")
	jtesting.AssertEqual(t, cfg.Profiles["blue"].Git.Email, "rock@example.com")
	jtesting.AssertEqual(t, cfg.CA.URL, "https://ca.example.com")
	jtesting.AssertEqual(t, cfg.CA.Provisioner, "myprovisioner")
}

func TestInitConfigMissingFile(t *testing.T) {
	err := initConfig("/nonexistent/path/to/config.yaml")
	jtesting.AssertError(t, err)
}

func TestInitConfigInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(configFile, []byte(":\tbad: yaml: [\x00"), 0o600)
	jtesting.AssertNoError(t, err)

	err = initConfig(configFile)
	jtesting.AssertError(t, err)
}

func TestInitConfigValidationFails(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(configFile, []byte("profiles: {}"), 0o600)
	jtesting.AssertNoError(t, err)

	cfg = Config{}
	err = initConfig(configFile)
	jtesting.AssertError(t, err)
}
