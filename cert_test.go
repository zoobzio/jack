//go:build testing

package jack

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jtesting "github.com/zoobzio/jack/testing"
)

// generateTestCert creates a self-signed ECDSA certificate with the given
// notBefore/notAfter window and writes the PEM-encoded cert to path.
func generateTestCert(t *testing.T, path string, notBefore, notAfter time.Time) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	jtesting.AssertNoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-agent"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	jtesting.AssertNoError(t, err)

	f, err := os.Create(path)
	jtesting.AssertNoError(t, err)
	defer f.Close()

	err = pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	jtesting.AssertNoError(t, err)
}

// setupAgentDir creates <tmpDir>/agents/<agent>/ and returns tmpDir.
func setupAgentDir(t *testing.T, agent string) string {
	t.Helper()
	tmpDir := t.TempDir()
	env = Env{ConfigDir: tmpDir}
	agentDir := filepath.Join(tmpDir, "agents", agent)
	err := os.MkdirAll(agentDir, 0o750)
	jtesting.AssertNoError(t, err)
	return tmpDir
}

// --- certPath ---

func TestCertPath(t *testing.T) {
	tmpDir := t.TempDir()
	env = Env{ConfigDir: tmpDir}
	got := certPath("blue")
	want := filepath.Join(tmpDir, "agents", "blue", "cert.pem")
	jtesting.AssertEqual(t, got, want)
}

// --- keyPath ---

func TestKeyPath(t *testing.T) {
	tmpDir := t.TempDir()
	env = Env{ConfigDir: tmpDir}
	got := keyPath("blue")
	want := filepath.Join(tmpDir, "agents", "blue", "key.pem")
	jtesting.AssertEqual(t, got, want)
}

// --- hasCert ---

func TestHasCertFalseWhenMissing(t *testing.T) {
	setupAgentDir(t, "blue")
	jtesting.AssertEqual(t, hasCert("blue"), false)
}

func TestHasCertTrueWhenPresent(t *testing.T) {
	tmpDir := setupAgentDir(t, "blue")
	certFile := filepath.Join(tmpDir, "agents", "blue", "cert.pem")
	err := os.WriteFile(certFile, []byte("dummy"), 0o600)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, hasCert("blue"), true)
}

// --- certExpiry ---

func TestCertExpiryParsesNotAfter(t *testing.T) {
	tmpDir := setupAgentDir(t, "blue")
	certFile := filepath.Join(tmpDir, "agents", "blue", "cert.pem")

	notBefore := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	notAfter := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	generateTestCert(t, certFile, notBefore, notAfter)

	expiry, err := certExpiry("blue")
	jtesting.AssertNoError(t, err)
	// x509 truncates to seconds.
	jtesting.AssertEqual(t, expiry.UTC().Truncate(time.Second), notAfter)
}

func TestCertExpiryErrorWhenMissing(t *testing.T) {
	setupAgentDir(t, "blue")
	_, err := certExpiry("blue")
	jtesting.AssertError(t, err)
}

func TestCertExpiryErrorOnInvalidPEM(t *testing.T) {
	tmpDir := setupAgentDir(t, "blue")
	certFile := filepath.Join(tmpDir, "agents", "blue", "cert.pem")
	err := os.WriteFile(certFile, []byte("not a pem block"), 0o600)
	jtesting.AssertNoError(t, err)

	_, err = certExpiry("blue")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "no PEM block"), true)
}

// --- certNeedsRenewal ---

func TestCertNeedsRenewalTrueWhenNoCert(t *testing.T) {
	setupAgentDir(t, "blue")
	jtesting.AssertEqual(t, certNeedsRenewal("blue", 7*24*time.Hour), true)
}

func TestCertNeedsRenewalTrueWhenExpiringWithinThreshold(t *testing.T) {
	tmpDir := setupAgentDir(t, "blue")
	certFile := filepath.Join(tmpDir, "agents", "blue", "cert.pem")

	// Cert expires in 1 hour — within a 7-day threshold.
	generateTestCert(t, certFile, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))

	jtesting.AssertEqual(t, certNeedsRenewal("blue", 7*24*time.Hour), true)
}

func TestCertNeedsRenewalFalseWhenFarFromExpiry(t *testing.T) {
	tmpDir := setupAgentDir(t, "blue")
	certFile := filepath.Join(tmpDir, "agents", "blue", "cert.pem")

	// Cert expires in 30 days — outside a 7-day threshold.
	generateTestCert(t, certFile, time.Now().Add(-time.Hour), time.Now().Add(30*24*time.Hour))

	jtesting.AssertEqual(t, certNeedsRenewal("blue", 7*24*time.Hour), false)
}

func TestCertExpiryErrorOnInvalidDER(t *testing.T) {
	tmpDir := setupAgentDir(t, "blue")
	certFile := filepath.Join(tmpDir, "agents", "blue", "cert.pem")
	// Write a PEM block with invalid DER bytes.
	invalidPEM := "-----BEGIN CERTIFICATE-----\nZm9vYmFy\n-----END CERTIFICATE-----\n"
	err := os.WriteFile(certFile, []byte(invalidPEM), 0o600)
	jtesting.AssertNoError(t, err)

	_, err = certExpiry("blue")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "parsing cert"), true)
}

func TestCertNeedsRenewalTrueWhenAlreadyExpired(t *testing.T) {
	tmpDir := setupAgentDir(t, "blue")
	certFile := filepath.Join(tmpDir, "agents", "blue", "cert.pem")

	// Cert expired 1 hour ago.
	generateTestCert(t, certFile, time.Now().Add(-2*time.Hour), time.Now().Add(-time.Hour))

	jtesting.AssertEqual(t, certNeedsRenewal("blue", 7*24*time.Hour), true)
}
