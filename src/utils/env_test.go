package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempEnv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	return path
}

func TestLoadDotEnv(t *testing.T) {
	os.Unsetenv("TEST_A")
	os.Unsetenv("TEST_B")
	os.Unsetenv("TEST_C")

	path := writeTempEnv(t, `
# comment line
TEST_A=hello
export TEST_B="quoted value"
TEST_C=spaced value  
`)
	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv failed: %v", err)
	}

	if got := os.Getenv("TEST_A"); got != "hello" {
		t.Errorf("TEST_A = %q, want %q", got, "hello")
	}
	if got := os.Getenv("TEST_B"); got != "quoted value" {
		t.Errorf("TEST_B = %q, want %q", got, "quoted value")
	}
	if got := os.Getenv("TEST_C"); got != "spaced value" {
		t.Errorf("TEST_C = %q, want %q", got, "spaced value")
	}
}

func TestLoadDotEnvDoesNotOverrideExisting(t *testing.T) {
	os.Setenv("TEST_EXISTING", "from-shell")
	defer os.Unsetenv("TEST_EXISTING")

	path := writeTempEnv(t, "TEST_EXISTING=from-env-file\n")
	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv failed: %v", err)
	}

	if got := os.Getenv("TEST_EXISTING"); got != "from-shell" {
		t.Errorf("existing env var was overwritten: %q", got)
	}
}

func TestLoadDotEnvMissingFileIsOK(t *testing.T) {
	if err := LoadDotEnv(filepath.Join(t.TempDir(), "does-not-exist")); err != nil {
		t.Errorf("missing .env should be optional, got error: %v", err)
	}
}

func TestLoadDotEnvMalformedLine(t *testing.T) {
	path := writeTempEnv(t, "TEST_BAD_NO_EQUALS\n")
	if err := LoadDotEnv(path); err == nil {
		t.Error("malformed line should return an error")
	}
}
