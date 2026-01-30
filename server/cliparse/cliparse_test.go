// Copyright (c) 2025 Daniel Kuo.
// Source-available; no permission granted to use, copy, modify, or distribute. See LICENSE.

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

func TestParseFlags_MissingDatabaseURL(t *testing.T) {
	os.Clearenv()
	os.Setenv("ADMIN_KEY_SALT", "test-salt")
	os.Setenv("POLL_SLUG_SALT", "test-slug")
	defer os.Clearenv()

	_, err := ParseFlags([]string{})
	if err == nil {
		t.Error("Expected error for missing DATABASE_URL")
	}
	if err.Error() != "DATABASE_URL required" {
		t.Errorf("Expected 'DATABASE_URL required' error, got: %v", err)
	}
}

func TestParseFlags_MissingAdminKeySalt(t *testing.T) {
	os.Clearenv()
	os.Setenv("DATABASE_URL", "postgres://test")
	os.Setenv("POLL_SLUG_SALT", "test-slug")
	defer os.Clearenv()

	_, err := ParseFlags([]string{})
	if err == nil {
		t.Error("Expected error for missing ADMIN_KEY_SALT")
	}
	if err.Error() != "ADMIN_KEY_SALT required" {
		t.Errorf("Expected 'ADMIN_KEY_SALT required' error, got: %v", err)
	}
}

func TestParseFlags_MissingPollSlugSalt(t *testing.T) {
	os.Clearenv()
	os.Setenv("DATABASE_URL", "postgres://test")
	os.Setenv("ADMIN_KEY_SALT", "test-salt")
	defer os.Clearenv()

	_, err := ParseFlags([]string{})
	if err == nil {
		t.Error("Expected error for missing POLL_SLUG_SALT")
	}
	if err.Error() != "POLL_SLUG_SALT required" {
		t.Errorf("Expected 'POLL_SLUG_SALT required' error, got: %v", err)
	}
}

func TestParseFlags_InvalidPort(t *testing.T) {
	os.Clearenv()
	os.Setenv("PORT", "not-a-number")
	os.Setenv("DATABASE_URL", "postgres://test")
	os.Setenv("ADMIN_KEY_SALT", "test-salt")
	os.Setenv("POLL_SLUG_SALT", "test-slug")
	defer os.Clearenv()

	_, err := ParseFlags([]string{})
	if err == nil {
		t.Error("Expected error for invalid PORT")
	}
	if err.Error() != "invalid PORT env variable" {
		t.Errorf("Expected 'invalid PORT env variable' error, got: %v", err)
	}
}

func TestParseFlags_DefaultPort(t *testing.T) {
	os.Clearenv()
	os.Setenv("DATABASE_URL", "postgres://test")
	os.Setenv("ADMIN_KEY_SALT", "test-salt")
	os.Setenv("POLL_SLUG_SALT", "test-slug")
	defer os.Clearenv()

	cfg, err := ParseFlags([]string{})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Port != 3318 {
		t.Errorf("Expected default port 3318, got %d", cfg.Port)
	}
}

func TestParseFlags_InvalidCLIFlag(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	_, err := ParseFlags([]string{"--invalid-flag"})
	if err == nil {
		t.Error("Expected error for invalid CLI flag")
	}
}

func TestParseFlags_AllEnvVarsNoFlags(t *testing.T) {
	os.Clearenv()
	os.Setenv("PORT", "4000")
	os.Setenv("DATABASE_URL", "postgres://mydb")
	os.Setenv("ADMIN_KEY_SALT", "admin-secret")
	os.Setenv("POLL_SLUG_SALT", "slug-secret")
	defer os.Clearenv()

	cfg, err := ParseFlags([]string{})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Port != 4000 {
		t.Errorf("Expected port 4000, got %d", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://mydb" {
		t.Errorf("Expected DatabaseURL 'postgres://mydb', got '%s'", cfg.DatabaseURL)
	}
	if cfg.AdminKeySalt != "admin-secret" {
		t.Errorf("Expected AdminKeySalt 'admin-secret', got '%s'", cfg.AdminKeySalt)
	}
	if cfg.PollSlugSalt != "slug-secret" {
		t.Errorf("Expected PollSlugSalt 'slug-secret', got '%s'", cfg.PollSlugSalt)
	}
}

func TestParseFlags_AllCLIFlags(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	cfg, err := ParseFlags([]string{
		"-p", "5000",
		"-d", "postgres://clidb",
		"-admin-salt", "cli-admin",
		"-slug-salt", "cli-slug",
	})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Port != 5000 {
		t.Errorf("Expected port 5000, got %d", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://clidb" {
		t.Errorf("Expected DatabaseURL 'postgres://clidb', got '%s'", cfg.DatabaseURL)
	}
	if cfg.AdminKeySalt != "cli-admin" {
		t.Errorf("Expected AdminKeySalt 'cli-admin', got '%s'", cfg.AdminKeySalt)
	}
	if cfg.PollSlugSalt != "cli-slug" {
		t.Errorf("Expected PollSlugSalt 'cli-slug', got '%s'", cfg.PollSlugSalt)
	}
}

func TestParseFlags_PartialCLIOverride(t *testing.T) {
	os.Clearenv()
	os.Setenv("PORT", "9000")
	os.Setenv("DATABASE_URL", "postgres://envdb")
	os.Setenv("ADMIN_KEY_SALT", "env-admin")
	os.Setenv("POLL_SLUG_SALT", "env-slug")
	defer os.Clearenv()

	// Only override port and database via CLI
	cfg, err := ParseFlags([]string{"-p", "7000", "-d", "postgres://clidb"})
	if err != nil {
		t.Fatal(err)
	}

	// CLI values should override
	if cfg.Port != 7000 {
		t.Errorf("Expected port 7000 (from CLI), got %d", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://clidb" {
		t.Errorf("Expected DatabaseURL from CLI, got '%s'", cfg.DatabaseURL)
	}

	// Env values should remain for non-overridden flags
	if cfg.AdminKeySalt != "env-admin" {
		t.Errorf("Expected AdminKeySalt from env, got '%s'", cfg.AdminKeySalt)
	}
	if cfg.PollSlugSalt != "env-slug" {
		t.Errorf("Expected PollSlugSalt from env, got '%s'", cfg.PollSlugSalt)
	}
}
