package leekwars

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ResolveToken cherche le token LeekWars dans l'ordre : variable LEEKWARS_TOKEN,
// fichier désigné par LEEKWARS_KEY_FILE, puis fichier defaultPath. Renvoie une
// chaîne vide, sans erreur, si rien n'est configuré.
func ResolveToken(defaultPath string) (string, error) {
	if tok := strings.TrimSpace(os.Getenv("LEEKWARS_TOKEN")); tok != "" {
		return tok, nil
	}
	if path := os.Getenv("LEEKWARS_KEY_FILE"); path != "" {
		return readTokenFile(path)
	}
	tok, err := readTokenFile(defaultPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return tok, err
}

func readTokenFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("lecture du fichier de token %s : %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}
