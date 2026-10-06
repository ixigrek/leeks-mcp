package summary

import (
	"encoding/json"
	"math"
	"strconv"
)

// flexFloat accepte un nombre JSON ou une chaîne numérique (l'API renvoie
// parfois "ratio": "2.23", et "∞" pour un poireau sans défaite).
type flexFloat float64

func (f *flexFloat) UnmarshalJSON(data []byte) error {
	s := string(data)
	if len(data) > 0 && data[0] == '"' {
		// Décodage JSON complet : l'API échappe "∞" en "\u221e".
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
	}
	switch s {
	case "", "null":
		*f = 0
		return nil
	case "∞", "Infinity", "inf", "+Inf":
		*f = flexFloat(math.Inf(1))
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*f = flexFloat(v)
	return nil
}

// Ratio est un ratio victoires/défaites ; l'infini (aucune défaite) se
// sérialise en "∞" comme dans l'API, car JSON n'a pas d'infini.
type Ratio float64

func (r Ratio) MarshalJSON() ([]byte, error) {
	if math.IsInf(float64(r), 0) {
		return []byte(`"∞"`), nil
	}
	return []byte(strconv.FormatFloat(float64(r), 'f', -1, 64)), nil
}
