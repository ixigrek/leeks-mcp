package leekwars

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTokenPrefersEnvVariable(t *testing.T) {
	t.Setenv("LEEKWARS_TOKEN", "  tok-env \n")
	t.Setenv("LEEKWARS_KEY_FILE", "")
	got, err := ResolveToken("/chemin/inexistant/key")
	if err != nil {
		t.Fatal(err)
	}
	if got != "tok-env" {
		t.Fatalf("token = %q, attendu tok-env", got)
	}
}

func TestResolveTokenReadsKeyFileFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cle")
	if err := os.WriteFile(path, []byte("tok-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LEEKWARS_TOKEN", "")
	t.Setenv("LEEKWARS_KEY_FILE", path)
	got, err := ResolveToken("/chemin/inexistant/key")
	if err != nil {
		t.Fatal(err)
	}
	if got != "tok-file" {
		t.Fatalf("token = %q, attendu tok-file", got)
	}
}

func TestResolveTokenFallsBackToDefaultFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key")
	if err := os.WriteFile(path, []byte("tok-default"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LEEKWARS_TOKEN", "")
	t.Setenv("LEEKWARS_KEY_FILE", "")
	got, err := ResolveToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "tok-default" {
		t.Fatalf("token = %q, attendu tok-default", got)
	}
}

func TestResolveTokenReturnsEmptyWhenNothingConfigured(t *testing.T) {
	t.Setenv("LEEKWARS_TOKEN", "")
	t.Setenv("LEEKWARS_KEY_FILE", "")
	got, err := ResolveToken(filepath.Join(t.TempDir(), "absent"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("token = %q, attendu vide", got)
	}
}
