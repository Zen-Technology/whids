package server

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xrawsec/whids/utils"
)

// TestMain prepares the fixtures the manager tests expect under ./data.
// The upstream project relied on a developer-local data directory that was
// never committed, which made these tests impossible to run in CI. Generating
// a throw-away self-signed certificate here keeps the suite self-contained.
func TestMain(m *testing.M) {
	// Start from a clean slate: a database left behind by an interrupted run
	// keeps the previous admin API key and makes every admin test fail with 403.
	for _, p := range []string{mconf.Database, mconf.Logging.Root, mconf.DumpDir, fconf.Logging.Dir} {
		os.RemoveAll(p)
	}

	if err := ensureTLSFixtures(mconf.TLS.Cert, mconf.TLS.Key); err != nil {
		fmt.Fprintf(os.Stderr, "failed to prepare TLS fixtures: %s\n", err)
		os.Exit(1)
	}

	fp, err := utils.CertFileSha256(mconf.TLS.Cert)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fingerprint TLS fixture: %s\n", err)
		os.Exit(1)
	}

	// rebuild client configs now that the pinned fingerprint is known
	testServerFingerprint = fp
	cconf = makeClientConfig(&mconf)
	fconf = makeForwarderConfig(&mconf)

	os.Exit(m.Run())
}

func ensureTLSFixtures(certPath, keyPath string) error {
	if fileExists(certPath) && fileExists(keyPath) {
		return nil
	}

	for _, p := range []string{certPath, keyPath} {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			return err
		}
	}

	key, cert, err := utils.GenerateCert("whids-test", []string{"localhost", "127.0.0.1"}, 24*time.Hour)
	if err != nil {
		return err
	}

	if err := os.WriteFile(keyPath, key, 0o600); err != nil {
		return err
	}
	return os.WriteFile(certPath, cert, 0o600)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
