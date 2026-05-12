//go:build testing

package jack

import (
	"fmt"
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestDoSyncCredentialsSkipsNonDarwin(t *testing.T) {
	called := false
	read := func() ([]byte, error) {
		called = true
		return nil, nil
	}
	err := doSyncCredentials("linux", read, nil)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, called, false)
}

func TestDoSyncCredentialsReadError(t *testing.T) {
	read := func() ([]byte, error) {
		return nil, fmt.Errorf("keychain locked")
	}
	err := doSyncCredentials("darwin", read, nil)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "keychain locked"), true)
}

func TestDoSyncCredentialsEmptyCredentials(t *testing.T) {
	read := func() ([]byte, error) {
		return []byte("   \n  "), nil
	}
	err := doSyncCredentials("darwin", read, nil)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "empty credentials"), true)
}

func TestDoSyncCredentialsWriteError(t *testing.T) {
	read := func() ([]byte, error) {
		return []byte(`{"token":"abc"}`), nil
	}
	write := func(_ string, _ []byte) error {
		return fmt.Errorf("disk full")
	}
	err := doSyncCredentials("darwin", read, write)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "writing credentials"), true)
}

func TestDoSyncCredentialsSuccess(t *testing.T) {
	var writtenPath string
	var writtenData []byte

	read := func() ([]byte, error) {
		return []byte("  {\"token\":\"abc\"}  \n"), nil
	}
	write := func(path string, data []byte) error {
		writtenPath = path
		writtenData = data
		return nil
	}

	err := doSyncCredentials("darwin", read, write)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, strings.HasSuffix(writtenPath, ".claude/.credentials.json"), true)
	jtesting.AssertEqual(t, string(writtenData), `{"token":"abc"}`)
}
