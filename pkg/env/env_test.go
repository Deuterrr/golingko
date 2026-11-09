package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGet(t *testing.T) {
	_ = os.Setenv("TEST_STR", "hello")
	_ = os.Setenv("TEST_INT", "42")
	_ = os.Setenv("TEST_INT64", "9876543210")
	_ = os.Setenv("TEST_BOOL", "true")
	_ = os.Setenv("TEST_FLOAT", "3.14")
	defer func() {
		_ = os.Unsetenv("TEST_STR")
		_ = os.Unsetenv("TEST_INT")
		_ = os.Unsetenv("TEST_INT64")
		_ = os.Unsetenv("TEST_BOOL")
		_ = os.Unsetenv("TEST_FLOAT")
	}()

	if v := Get("TEST_STR", "fallback"); v != "hello" {
		t.Errorf("expected hello, got %s", v)
	}
	if v := Get("TEST_INT", 10); v != 42 {
		t.Errorf("expected 42, got %d", v)
	}
	if v := Get("TEST_INT64", int64(100)); v != 9876543210 {
		t.Errorf("expected 9876543210, got %d", v)
	}
	if v := Get("TEST_BOOL", false); v != true {
		t.Errorf("expected true, got %v", v)
	}
	if v := Get("TEST_FLOAT", 1.0); v != 3.14 {
		t.Errorf("expected 3.14, got %f", v)
	}

	// Default fallbacks
	if v := Get("NON_EXISTENT_KEY", "fallback"); v != "fallback" {
		t.Errorf("expected fallback, got %s", v)
	}
	if v := Get("NON_EXISTENT_INT", 99); v != 99 {
		t.Errorf("expected 99, got %d", v)
	}
}

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env.test")
	content := []byte("LOAD_TEST_A=apple\n# comment\nLOAD_TEST_B=\"banana\"\nLOAD_TEST_C='cherry'\n")
	if err := os.WriteFile(envPath, content, 0644); err != nil {
		t.Fatalf("failed to create temp env file: %v", err)
	}

	_ = os.Unsetenv("LOAD_TEST_A")
	_ = os.Unsetenv("LOAD_TEST_B")
	_ = os.Unsetenv("LOAD_TEST_C")
	defer func() {
		_ = os.Unsetenv("LOAD_TEST_A")
		_ = os.Unsetenv("LOAD_TEST_B")
		_ = os.Unsetenv("LOAD_TEST_C")
	}()

	Load(envPath)

	if v := os.Getenv("LOAD_TEST_A"); v != "apple" {
		t.Errorf("expected apple, got %q", v)
	}
	if v := os.Getenv("LOAD_TEST_B"); v != "banana" {
		t.Errorf("expected banana, got %q", v)
	}
	if v := os.Getenv("LOAD_TEST_C"); v != "cherry" {
		t.Errorf("expected cherry, got %q", v)
	}
}
