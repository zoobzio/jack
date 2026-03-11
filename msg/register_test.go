//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunRegisterSuccess(t *testing.T) {
	registerer := func(user, pass, token string) (*Registration, error) {
		return &Registration{
			UserID:      "@" + user + ":localhost",
			AccessToken: "tok_abc",
		}, nil
	}

	err := runRegister("agent", "pass", "jack", registerer)
	jtesting.AssertNoError(t, err)
}

func TestRunRegisterError(t *testing.T) {
	registerer := func(_, _, _ string) (*Registration, error) {
		return nil, fmt.Errorf("registration failed")
	}
	err := runRegister("agent", "pass", "jack", registerer)
	jtesting.AssertError(t, err)
}
