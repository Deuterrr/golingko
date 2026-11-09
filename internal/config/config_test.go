package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigDefaults(t *testing.T) {
	// Clear specific env keys that might interfere
	keys := []string{
		"SERVER_ADDR", "ADDR", "PORT", "SERVER_PORT", "HTTP_ROUTER", "ROUTER_ENGINE",
		"DATABASE_URL", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"REDIS_URL", "MIN_TAP_BALANCE", "JAKLINGKO_FARE_CAP", "JOURNEY_EXPIRY_HOURS",
		"LOG_DRIVER", "LOG_VERBOSITY", "LOG_LEVEL", "LOG_FORMAT",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}

	cfg := Load()

	if cfg.ServerAddr != "127.0.0.1:8080" {
		t.Errorf("expected ServerAddr 127.0.0.1:8080, got %s", cfg.ServerAddr)
	}
	if cfg.HTTPRouter != "nethttp" {
		t.Errorf("expected HTTPRouter nethttp, got %s", cfg.HTTPRouter)
	}
	if cfg.DB.Host != "localhost" || cfg.DB.Port != 5432 {
		t.Errorf("unexpected DB host/port: %+v", cfg.DB)
	}
	if cfg.MinTapBalance != 5000 {
		t.Errorf("expected MinTapBalance 5000, got %d", cfg.MinTapBalance)
	}
	if cfg.JakLingkoFareCap != 10000 {
		t.Errorf("expected JakLingkoFareCap 10000, got %d", cfg.JakLingkoFareCap)
	}
	if cfg.JourneyExpiryHours != 4*time.Hour {
		t.Errorf("expected JourneyExpiryHours 4h, got %v", cfg.JourneyExpiryHours)
	}
	if cfg.Logging.Driver != "slog" || cfg.Logging.Verbosity != "info" || cfg.Logging.Format != "json" {
		t.Errorf("unexpected Logging config: %+v", cfg.Logging)
	}
}

func TestConfigEnvOverrides(t *testing.T) {
	t.Setenv("SERVER_ADDR", "0.0.0.0:9090")
	t.Setenv("HTTP_ROUTER", "gin")
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("MIN_TAP_BALANCE", "7500")
	t.Setenv("LOG_DRIVER", "zap")
	t.Setenv("LOG_VERBOSITY", "debug")

	cfg := Load()

	if cfg.ServerAddr != "0.0.0.0:9090" {
		t.Errorf("expected ServerAddr 0.0.0.0:9090, got %s", cfg.ServerAddr)
	}
	if cfg.HTTPRouter != "gin" {
		t.Errorf("expected HTTPRouter gin, got %s", cfg.HTTPRouter)
	}
	if cfg.DB.Host != "db.internal" || cfg.DB.Port != 5433 {
		t.Errorf("unexpected DB host/port: %+v", cfg.DB)
	}
	if cfg.MinTapBalance != 7500 {
		t.Errorf("expected MinTapBalance 7500, got %d", cfg.MinTapBalance)
	}
	if cfg.Logging.Driver != "zap" {
		t.Errorf("expected Logging.Driver zap, got %s", cfg.Logging.Driver)
	}
	if cfg.Logging.Verbosity != "debug" {
		t.Errorf("expected Logging.Verbosity debug, got %s", cfg.Logging.Verbosity)
	}
}

func TestLoadEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env.test")

	content := `
# Comment line
TEST_GOLINGKO_VAR1=hello
TEST_GOLINGKO_VAR2="quoted value"
INVALID_LINE_WITHOUT_EQUALS
`
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test env file: %v", err)
	}

	loadEnvFile(envPath)

	if val := os.Getenv("TEST_GOLINGKO_VAR1"); val != "hello" {
		t.Errorf("expected hello, got %q", val)
	}
	if val := os.Getenv("TEST_GOLINGKO_VAR2"); val != "quoted value" {
		t.Errorf("expected quoted value, got %q", val)
	}

	// Should not overwrite existing environment variable
	t.Setenv("TEST_GOLINGKO_EXISTING", "original")
	overridePath := filepath.Join(tmpDir, ".env.override")
	_ = os.WriteFile(overridePath, []byte("TEST_GOLINGKO_EXISTING=overwritten"), 0600)
	loadEnvFile(overridePath)

	if val := os.Getenv("TEST_GOLINGKO_EXISTING"); val != "original" {
		t.Errorf("expected original, got %q", val)
	}
}

func TestConfigDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://postgres.myref:secretpass@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?sslmode=require")

	cfg := Load()

	if cfg.DB.URL != "postgresql://postgres.myref:secretpass@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?sslmode=require" {
		t.Errorf("expected matching URL, got %s", cfg.DB.URL)
	}
	if cfg.DB.Host != "aws-0-ap-southeast-1.pooler.supabase.com" {
		t.Errorf("expected Host aws-0-ap-southeast-1.pooler.supabase.com, got %s", cfg.DB.Host)
	}
	if cfg.DB.Port != 6543 {
		t.Errorf("expected Port 6543, got %d", cfg.DB.Port)
	}
	if cfg.DB.User != "postgres.myref" {
		t.Errorf("expected User postgres.myref, got %s", cfg.DB.User)
	}
	if cfg.DB.Password != "secretpass" {
		t.Errorf("expected Password secretpass, got %s", cfg.DB.Password)
	}
	if cfg.DB.DBName != "postgres" {
		t.Errorf("expected DBName postgres, got %s", cfg.DB.DBName)
	}
	if cfg.DB.SSLMode != "require" {
		t.Errorf("expected SSLMode require, got %s", cfg.DB.SSLMode)
	}
}

func TestConfigPortFallback(t *testing.T) {
	t.Setenv("SERVER_ADDR", "")
	t.Setenv("ADDR", "")
	t.Setenv("PORT", "3000")

	cfg := Load()

	if cfg.ServerAddr != "127.0.0.1:3000" {
		t.Errorf("expected ServerAddr 127.0.0.1:3000, got %s", cfg.ServerAddr)
	}
}
