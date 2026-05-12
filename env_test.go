//go:build testing

package jack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestLoadEnvDefaults(t *testing.T) {
	e := loadEnv()
	jtesting.AssertEqual(t, e.ConfigDir, "~/.config/jack")
	jtesting.AssertEqual(t, e.DataDir, "~/.jack")
}

func TestLoadEnvRespectsJACK_CONFIG_DIR(t *testing.T) {
	t.Setenv("JACK_CONFIG_DIR", "/custom/config")
	e := loadEnv()
	jtesting.AssertEqual(t, e.ConfigDir, "/custom/config")
	jtesting.AssertEqual(t, e.DataDir, "~/.jack")
}

func TestLoadEnvRespectsJACK_DATA_DIR(t *testing.T) {
	t.Setenv("JACK_DATA_DIR", "/custom/data")
	e := loadEnv()
	jtesting.AssertEqual(t, e.ConfigDir, "~/.config/jack")
	jtesting.AssertEqual(t, e.DataDir, "/custom/data")
}

func TestLoadEnvRespectsBothVars(t *testing.T) {
	t.Setenv("JACK_CONFIG_DIR", "/my/config")
	t.Setenv("JACK_DATA_DIR", "/my/data")
	e := loadEnv()
	jtesting.AssertEqual(t, e.ConfigDir, "/my/config")
	jtesting.AssertEqual(t, e.DataDir, "/my/data")
}

func TestExpandHome(t *testing.T) {
	t.Run("expands tilde prefix", func(t *testing.T) {
		home, err := os.UserHomeDir()
		jtesting.AssertNoError(t, err)

		got := expandHome("~/foo/bar")
		want := filepath.Join(home, "foo/bar")
		jtesting.AssertEqual(t, got, want)
	})

	t.Run("tilde only", func(t *testing.T) {
		home, err := os.UserHomeDir()
		jtesting.AssertNoError(t, err)

		got := expandHome("~")
		jtesting.AssertEqual(t, got, home)
	})

	t.Run("leaves absolute paths alone", func(t *testing.T) {
		jtesting.AssertEqual(t, expandHome("/absolute/path"), "/absolute/path")
	})

	t.Run("leaves relative paths alone", func(t *testing.T) {
		jtesting.AssertEqual(t, expandHome("relative/path"), "relative/path")
	})

	t.Run("leaves empty string alone", func(t *testing.T) {
		jtesting.AssertEqual(t, expandHome(""), "")
	})
}

func TestEnvConfigDir(t *testing.T) {
	e := Env{ConfigDir: "~/.config/jack"}
	home, err := os.UserHomeDir()
	jtesting.AssertNoError(t, err)

	got := e.configDir()
	jtesting.AssertEqual(t, got, filepath.Join(home, ".config/jack"))
}

func TestEnvConfigDirAbsolute(t *testing.T) {
	e := Env{ConfigDir: "/tmp/jack-config"}
	jtesting.AssertEqual(t, e.configDir(), "/tmp/jack-config")
}

func TestEnvConfigPath(t *testing.T) {
	e := Env{ConfigDir: "/tmp/jack-config"}
	got := e.configPath()
	jtesting.AssertEqual(t, got, "/tmp/jack-config/config.yaml")
	jtesting.AssertEqual(t, strings.HasSuffix(got, "config.yaml"), true)
}

func TestEnvDataDir(t *testing.T) {
	e := Env{DataDir: "~/.jack"}
	home, err := os.UserHomeDir()
	jtesting.AssertNoError(t, err)

	got := e.dataDir()
	jtesting.AssertEqual(t, got, filepath.Join(home, ".jack"))
}

func TestEnvDataDirAbsolute(t *testing.T) {
	e := Env{DataDir: "/tmp/jack-data"}
	jtesting.AssertEqual(t, e.dataDir(), "/tmp/jack-data")
}
