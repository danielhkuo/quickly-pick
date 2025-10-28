// cliparse/cliparse_test.go
package cliparse

import (
	"os"
	"testing"
)

func TestParseFlags_EnvVars(t *testing.T) {
	// Set env vars
	os.Setenv("PORT", "9000")
	os.Setenv("DATABASE_URL", "postgres://test")
	os.Setenv("ADMIN_KEY_SALT", "test-salt")
	os.Setenv("POLL_SLUG_SALT", "test-slug")
	defer os.Clearenv()

	cfg, err := ParseFlags([]string{})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Port != 9000 {
		t.Errorf("expected port 9000, got %d", cfg.Port)
	}
}

func TestParseFlags_CLIOverridesEnv(t *testing.T) {
	os.Setenv("PORT", "9000")
	defer os.Clearenv()

	cfg, err := ParseFlags([]string{"-p", "8080", "-d", "file:test.db", "-admin-salt", "s1", "-slug-salt", "s2"})
	if err != nil {
		t.Fatal(err)
	}

	// CLI should override env
	if cfg.Port != 8080 {
		t.Errorf("CLI should override env: expected 8080, got %d", cfg.Port)
	}
}
