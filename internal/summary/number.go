package summary

import (
	"strconv"
	"strings"
)

// flexFloat accepte un nombre JSON ou une chaîne numérique (l'API renvoie
// parfois "ratio": "2.23").
type flexFloat float64

func (f *flexFloat) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*f = flexFloat(v)
	return nil
}
