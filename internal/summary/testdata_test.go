package summary

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func items(t *testing.T) *leekwars.Items {
	t.Helper()
	it, err := leekwars.ParseItems(fixture(t, "weapons.json"), fixture(t, "chips.json"))
	if err != nil {
		t.Fatal(err)
	}
	return it
}
