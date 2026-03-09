//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func noopSaver(_, _ string) error { return nil }

func TestRunRegisterSuccess(t *testing.T) {
	var savedUser, savedToken string
	saver := func(user, token string) error {
		savedUser = user
		savedToken = token
		return nil
	}
	registerer := func(user, pass, token string) (*Registration, error) {
		return &Registration{
			UserID:      "@" + user + ":localhost",
			AccessToken: "tok_abc",
		}, nil
	}

	err := runRegister("agent", "pass", "jack", registerer, saver)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, savedUser, "agent")
	jtesting.AssertEqual(t, savedToken, "tok_abc")
}

func TestRunRegisterError(t *testing.T) {
	registerer := func(_, _, _ string) (*Registration, error) {
		return nil, fmt.Errorf("registration failed")
	}
	err := runRegister("agent", "pass", "jack", registerer, noopSaver)
	jtesting.AssertError(t, err)
}
