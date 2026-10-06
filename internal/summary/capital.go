package summary

import (
	"encoding/json"
	"fmt"
)

// costStep est un palier de conversion capital → points de caractéristique :
// à partir de Step points déjà achetés, Capital capital achète Sup points.
type costStep struct {
	Step, Capital, Sup int
}

// costs reprend la table du client officiel (src/model/costs.ts), miroir des
// constantes serveur.
var costs = map[string][]costStep{
	"life":       {{0, 1, 4}, {1000, 1, 3}, {2000, 1, 2}},
	"strength":   {{0, 1, 2}, {200, 1, 1}, {400, 2, 1}, {600, 3, 1}},
	"wisdom":     {{0, 1, 2}, {200, 1, 1}, {400, 2, 1}, {600, 3, 1}},
	"agility":    {{0, 1, 2}, {200, 1, 1}, {400, 2, 1}, {600, 3, 1}},
	"resistance": {{0, 1, 2}, {200, 1, 1}, {400, 2, 1}, {600, 3, 1}},
	"science":    {{0, 1, 2}, {200, 1, 1}, {400, 2, 1}, {600, 3, 1}},
	"magic":      {{0, 1, 2}, {200, 1, 1}, {400, 2, 1}, {600, 3, 1}},
	"frequency":  {{0, 1, 1}},
	"cores":      {{0, 20, 1}, {1, 30, 1}, {2, 40, 1}, {3, 50, 1}, {4, 60, 1}, {5, 70, 1}, {6, 80, 1}, {7, 90, 1}, {8, 100, 1}},
	"ram":        {{0, 20, 1}, {1, 30, 1}, {2, 40, 1}, {3, 50, 1}, {4, 60, 1}, {5, 70, 1}, {6, 80, 1}, {7, 90, 1}, {8, 100, 1}},
	"tp":         {{0, 30, 1}, {1, 35, 1}, {2, 40, 1}, {3, 45, 1}, {4, 50, 1}, {5, 55, 1}, {6, 60, 1}, {7, 65, 1}, {8, 70, 1}, {9, 75, 1}, {10, 80, 1}, {11, 85, 1}, {12, 90, 1}, {13, 95, 1}, {14, 100, 1}},
	"mp":         {{0, 20, 1}, {1, 40, 1}, {2, 60, 1}, {3, 80, 1}, {4, 100, 1}, {5, 120, 1}, {6, 140, 1}, {7, 160, 1}, {8, 180, 1}},
}

// IsStat indique si name est une caractéristique allouable.
func IsStat(name string) bool {
	_, ok := costs[name]
	return ok
}

// BaseStat est la valeur d'une caractéristique sans capital ni équipement.
func BaseStat(level int, stat string) int {
	switch stat {
	case "life":
		return 100 + (level-1)*3
	case "frequency":
		return 100
	case "cores":
		return 1
	case "ram":
		return 6
	case "tp":
		return 10
	case "mp":
		return 3
	}
	return 0
}

// TotalCapital est le capital total d'un poireau à ce niveau (formule serveur).
func TotalCapital(level int) int {
	capital := 50 + (level-1)*5 + level/100*45
	if level == 301 {
		capital += 95
	}
	return capital
}

// BonusToCapital est le capital qu'ont coûté bonus points d'une caractéristique
// (statBonusToCapital du client officiel) : c'est l'unité des stats d'un loadout.
func BonusToCapital(stat string, bonus int) int {
	steps := costs[stat]
	capital, total := 0, 0
	for total < bonus {
		i := 0
		for i < len(steps) && steps[i].Step <= total {
			i++
		}
		capital += steps[i-1].Capital
		total += steps[i-1].Sup
	}
	return capital
}

// CapitalSpent convertit les caractéristiques hors équipement d'un poireau en
// capital dépensé par stat, sans les stats à zéro.
func CapitalSpent(level int, stats map[string]int) map[string]int {
	out := map[string]int{}
	for stat := range costs {
		if c := BonusToCapital(stat, stats[stat]-BaseStat(level, stat)); c > 0 {
			out[stat] = c
		}
	}
	return out
}

// LeekBuild est l'état privé d'un poireau utile aux loadouts.
type LeekBuild struct {
	ID         int
	Name       string
	Level      int
	Capital    int
	MaxWeapons int
	Stats      map[string]int // caractéristiques hors équipement
	Weapons    []int          // templates (arme : item)
	Chips      []int          // templates
	Components []LoadoutComponent
}

// Build lit leek/get-private.
func Build(privateJSON []byte) (*LeekBuild, error) {
	var raw struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Level      int    `json:"level"`
		Capital    int    `json:"capital"`
		MaxWeapons int    `json:"max_weapons"`
		Weapons    []struct {
			Template int `json:"template"`
		} `json:"weapons"`
		Chips []struct {
			Template int `json:"template"`
		} `json:"chips"`
		Components []*struct {
			Template int             `json:"template"`
			Stats    json.RawMessage `json:"stats"`
		} `json:"components"`
	}
	if err := json.Unmarshal(privateJSON, &raw); err != nil {
		return nil, fmt.Errorf("leek/get-private : %w", err)
	}
	var flat map[string]json.RawMessage
	if err := json.Unmarshal(privateJSON, &flat); err != nil {
		return nil, fmt.Errorf("leek/get-private : %w", err)
	}
	b := &LeekBuild{
		ID: raw.ID, Name: raw.Name, Level: raw.Level, Capital: raw.Capital, MaxWeapons: raw.MaxWeapons,
		Stats: map[string]int{}, Weapons: []int{}, Chips: []int{}, Components: []LoadoutComponent{},
	}
	for _, name := range statNames {
		b.Stats[name] = intField(flat, name)
	}
	for _, w := range raw.Weapons {
		b.Weapons = append(b.Weapons, w.Template)
	}
	for _, c := range raw.Chips {
		b.Chips = append(b.Chips, c.Template)
	}
	// Les composants sont une liste de 8 emplacements, vides à null : l'index est la position.
	for i, c := range raw.Components {
		if c != nil {
			b.Components = append(b.Components, LoadoutComponent{Index: i, Template: c.Template, Stats: nullIfEmpty(c.Stats)})
		}
	}
	return b, nil
}

func nullIfEmpty(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("null")
	}
	return raw
}
