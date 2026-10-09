package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeEnvFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	return path
}

func TestLoadEnvFiles(t *testing.T) {
	t.Run("missing files are skipped", func(t *testing.T) {
		dir := t.TempDir()

		if err := loadEnvFiles(filepath.Join(dir, "nope.env"), filepath.Join(dir, "nope.envrc")); err != nil {
			t.Fatalf("loadEnvFiles() error = %v, want nil", err)
		}
	})

	t.Run("parses export prefix, quotes and comments", func(t *testing.T) {
		dir := t.TempDir()
		path := writeEnvFile(t, dir, ".env", strings.Join([]string{
			"# a comment",
			"",
			"export GREENLIGHT_TEST_PLAIN=plain",
			`GREENLIGHT_TEST_DQUOTED="double value"`,
			`GREENLIGHT_TEST_SQUOTED='single value'`,
			"  GREENLIGHT_TEST_PADDED = padded  ",
		}, "\n"))

		t.Cleanup(func() {
			os.Unsetenv("GREENLIGHT_TEST_PLAIN")
			os.Unsetenv("GREENLIGHT_TEST_DQUOTED")
			os.Unsetenv("GREENLIGHT_TEST_SQUOTED")
			os.Unsetenv("GREENLIGHT_TEST_PADDED")
		})

		if err := loadEnvFiles(path); err != nil {
			t.Fatalf("loadEnvFiles() error = %v, want nil", err)
		}

		want := map[string]string{
			"GREENLIGHT_TEST_PLAIN":   "plain",
			"GREENLIGHT_TEST_DQUOTED": "double value",
			"GREENLIGHT_TEST_SQUOTED": "single value",
			"GREENLIGHT_TEST_PADDED":  "padded",
		}
		for key, expected := range want {
			if got, ok := os.LookupEnv(key); !ok || got != expected {
				t.Errorf("%s = %q (set=%t), want %q", key, got, ok, expected)
			}
		}
	})

	t.Run("does not overwrite existing environment", func(t *testing.T) {
		dir := t.TempDir()
		path := writeEnvFile(t, dir, ".env", "GREENLIGHT_TEST_EXISTING=from-file\n")
		t.Setenv("GREENLIGHT_TEST_EXISTING", "from-env")

		if err := loadEnvFiles(path); err != nil {
			t.Fatalf("loadEnvFiles() error = %v, want nil", err)
		}

		if got := os.Getenv("GREENLIGHT_TEST_EXISTING"); got != "from-env" {
			t.Errorf("GREENLIGHT_TEST_EXISTING = %q, want %q", got, "from-env")
		}
	})

	t.Run("first file wins on duplicate keys", func(t *testing.T) {
		dir := t.TempDir()
		first := writeEnvFile(t, dir, ".env", "GREENLIGHT_TEST_DUP=first\n")
		second := writeEnvFile(t, dir, ".envrc", "GREENLIGHT_TEST_DUP=second\n")
		t.Cleanup(func() { os.Unsetenv("GREENLIGHT_TEST_DUP") })

		if err := loadEnvFiles(first, second); err != nil {
			t.Fatalf("loadEnvFiles() error = %v, want nil", err)
		}

		if got := os.Getenv("GREENLIGHT_TEST_DUP"); got != "first" {
			t.Errorf("GREENLIGHT_TEST_DUP = %q, want %q", got, "first")
		}
	})

	t.Run("malformed line reports path and line", func(t *testing.T) {
		dir := t.TempDir()
		path := writeEnvFile(t, dir, ".env", "GOOD=1\nthis is not a pair\n")

		err := loadEnvFiles(path)
		if err == nil {
			t.Fatal("loadEnvFiles() error = nil, want an error")
		}
		if !strings.Contains(err.Error(), ".env:2:") {
			t.Errorf("error = %q, want it to contain %q", err, ".env:2:")
		}
	})

	t.Run("invalid variable name is rejected", func(t *testing.T) {
		dir := t.TempDir()
		path := writeEnvFile(t, dir, ".env", "bad key=1\n")

		err := loadEnvFiles(path)
		if err == nil {
			t.Fatal("loadEnvFiles() error = nil, want an error")
		}
		if !strings.Contains(err.Error(), ".env:1:") {
			t.Errorf("error = %q, want it to contain %q", err, ".env:1:")
		}
	})
}
