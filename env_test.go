package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvSetsMissingValues(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	if err := os.Unsetenv("GOOGLE_API_KEY"); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GOOGLE_API_KEY=test-key\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("GOOGLE_API_KEY"); got != "test-key" {
		t.Fatalf("GOOGLE_API_KEY = %q, want test-key", got)
	}
}

func TestLoadDotEnvDoesNotOverrideExistingValues(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "shell-key")

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GOOGLE_API_KEY=file-key\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("GOOGLE_API_KEY"); got != "shell-key" {
		t.Fatalf("GOOGLE_API_KEY = %q, want shell-key", got)
	}
}

func TestLoadDotEnvSupportsQuotedValues(t *testing.T) {
	t.Setenv("QUOTED_ENV_TEST", "")
	if err := os.Unsetenv("QUOTED_ENV_TEST"); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("QUOTED_ENV_TEST=\"line one\\nline two\"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("QUOTED_ENV_TEST"); got != "line one\nline two" {
		t.Fatalf("QUOTED_ENV_TEST = %q, want quoted multiline value", got)
	}
}
