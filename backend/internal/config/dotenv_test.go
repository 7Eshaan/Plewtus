package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvDoesNotOverrideEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\nPLEWTUS_TEST_DOTENV_A=from_file\nexport PLEWTUS_TEST_DOTENV_B=\"quoted value\"\nPLEWTUS_TEST_DOTENV_C=from_file\nnot a pair\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PLEWTUS_TEST_DOTENV_C", "from_env")
	for _, k := range []string{"PLEWTUS_TEST_DOTENV_A", "PLEWTUS_TEST_DOTENV_B"} {
		k := k
		t.Cleanup(func() { os.Unsetenv(k) })
	}

	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("PLEWTUS_TEST_DOTENV_A"); got != "from_file" {
		t.Errorf("A = %q", got)
	}
	if got := os.Getenv("PLEWTUS_TEST_DOTENV_B"); got != "quoted value" {
		t.Errorf("B = %q", got)
	}
	if got := os.Getenv("PLEWTUS_TEST_DOTENV_C"); got != "from_env" {
		t.Errorf("real environment must win, got %q", got)
	}
	if err := loadDotEnv(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Errorf("missing file should be ignored: %v", err)
	}
}
