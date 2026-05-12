//go:build testing

package jack

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	jtesting "github.com/zoobzio/jack/testing"
)

// writeSelfSignedCert generates a self-signed certificate for the given agent
// with the specified validity window and writes it to the expected cert path.
func writeSelfSignedCert(t *testing.T, agent string, notBefore, notAfter time.Time) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: agent},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certFile := certPath(agent)
	if err := os.MkdirAll(filepath.Dir(certFile), 0o750); err != nil {
		t.Fatal(err)
	}

	f, err := os.Create(certFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatal(err)
	}
}

func TestRunRenewNoCAURL(t *testing.T) {
	cfg = Config{
		CA: CAConfig{URL: ""},
		Profiles: map[string]Profile{
			"blue": {},
		},
	}
	env = Env{ConfigDir: t.TempDir(), DataDir: t.TempDir()}

	err := runRenew(context.Background(), nil, func(_ context.Context, _ string) error {
		return nil
	})
	jtesting.AssertError(t, err)
}

func TestRunRenewDefaultsToAllAgents(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: t.TempDir()}
	cfg = Config{
		CA: CAConfig{URL: "https://ca.example.com"},
		Profiles: map[string]Profile{
			"blue": {},
			"red":  {},
		},
	}

	var renewed []string
	renewer := func(_ context.Context, agent string) error {
		renewed = append(renewed, agent)
		return nil
	}

	err := runRenew(context.Background(), nil, renewer)
	jtesting.AssertNoError(t, err)

	// Both agents should have been renewed (no existing certs → certNeedsRenewal = true).
	jtesting.AssertEqual(t, len(renewed), 2)
}

func TestRunRenewSkipsUnknownAgent(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: t.TempDir()}
	cfg = Config{
		CA: CAConfig{URL: "https://ca.example.com"},
		Profiles: map[string]Profile{
			"blue": {},
		},
	}

	var renewed []string
	renewer := func(_ context.Context, agent string) error {
		renewed = append(renewed, agent)
		return nil
	}

	// "ghost" is not in cfg.Profiles; it should be silently skipped.
	err := runRenew(context.Background(), []string{"ghost", "blue"}, renewer)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(renewed), 1)
	jtesting.AssertEqual(t, renewed[0], "blue")
}

func TestRunRenewSkipsFarFutureExpiry(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: t.TempDir()}
	cfg = Config{
		CA: CAConfig{URL: "https://ca.example.com"},
		Profiles: map[string]Profile{
			"blue": {},
		},
	}

	// Write a cert that expires far in the future — well outside the renewal threshold.
	writeSelfSignedCert(t, "blue", time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour))

	var renewed []string
	renewer := func(_ context.Context, agent string) error {
		renewed = append(renewed, agent)
		return nil
	}

	err := runRenew(context.Background(), []string{"blue"}, renewer)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(renewed), 0)
}

func TestRunRenewCallsRenewerForExpiring(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: t.TempDir()}
	cfg = Config{
		CA: CAConfig{URL: "https://ca.example.com"},
		Profiles: map[string]Profile{
			"blue": {},
		},
	}

	// Write a cert that is already expired — needs renewal.
	writeSelfSignedCert(t, "blue", time.Now().Add(-2*time.Hour), time.Now().Add(-time.Minute))

	var renewed []string
	renewer := func(_ context.Context, agent string) error {
		renewed = append(renewed, agent)
		return nil
	}

	err := runRenew(context.Background(), []string{"blue"}, renewer)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(renewed), 1)
	jtesting.AssertEqual(t, renewed[0], "blue")
}

func TestRunRenewReturnsRenewerError(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: t.TempDir()}
	cfg = Config{
		CA: CAConfig{URL: "https://ca.example.com"},
		Profiles: map[string]Profile{
			"blue": {},
		},
	}

	// No cert file → certNeedsRenewal returns true.
	renewer := func(_ context.Context, _ string) error {
		return errors.New("step: connection refused")
	}

	err := runRenew(context.Background(), []string{"blue"}, renewer)
	jtesting.AssertError(t, err)
}
