package summary

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
)

var statNames = []string{"life", "strength", "wisdom", "agility", "resistance", "science", "magic", "frequency", "cores", "ram", "tp", "mp"}

// WeaponRef identifie une arme équipée.
type WeaponRef struct {
	ID   int    `json:"id"`
	Item int    `json:"item"`
	Name string `json:"name"`
}

// ChipRef identifie une puce équipée.
type ChipRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// AIRef décrit l'IA assignée.
type AIRef struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Valid bool   `json:"valid"`
}

// Record est le bilan de combats.
type Record struct {
	Victories int     `json:"victories"`
	Draws     int     `json:"draws"`
	Defeats   int     `json:"defeats"`
	Ratio     float64 `json:"ratio"`
}

// Component est un composant équipé (données privées).
type Component struct {
	ID       int `json:"id"`
	Template int `json:"template"`
}

// LeekSummary est le résumé de leek/get (et de leek/get-private si fourni).
type LeekSummary struct {
	ID         int            `json:"id"`
	Name       string         `json:"name"`
	Level      int            `json:"level"`
	Talent     int            `json:"talent"`
	Farmer     int            `json:"farmer"`
	Stats      map[string]int `json:"stats"`
	TotalStats map[string]int `json:"total_stats"`
	Weapons    []WeaponRef    `json:"weapons"`
	Chips      []ChipRef      `json:"chips"`
	AI         *AIRef         `json:"ai"`
	Record     Record         `json:"record"`
	Fights     []FightLine    `json:"fights"`
	Capital    *int           `json:"capital,omitempty"`
	Spent      map[string]int `json:"capital_spent,omitempty"` // capital investi par stat, l'unité des loadouts
	Components []Component    `json:"components,omitempty"`
}

// rawLeek couvre les champs utiles de leek/get et leek/get-private.
type rawLeek struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Level  int    `json:"level"`
	Talent int    `json:"talent"`
	Farmer struct {
		ID int `json:"id"`
	} `json:"farmer"`
	Weapons []struct {
		ID       int `json:"id"`
		Template int `json:"template"`
	} `json:"weapons"`
	Chips []struct {
		ID       int `json:"id"`
		Template int `json:"template"`
	} `json:"chips"`
	AI *struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Valid bool   `json:"valid"`
	} `json:"ai"`
	Victories  int          `json:"victories"`
	Draws      int          `json:"draws"`
	Defeats    int          `json:"defeats"`
	Ratio      flexFloat    `json:"ratio"`
	Fights     []rawFight   `json:"fights"`
	Capital    *int         `json:"capital"`
	Components []*Component `json:"components"`
}

// Leek résume leek/get ; privateJSON (leek/get-private) est optionnel.
func Leek(publicJSON, privateJSON []byte, items *leekwars.Items) (*LeekSummary, error) {
	var raw rawLeek
	if err := json.Unmarshal(publicJSON, &raw); err != nil {
		return nil, fmt.Errorf("leek/get : %w", err)
	}
	var flat map[string]json.RawMessage
	if err := json.Unmarshal(publicJSON, &flat); err != nil {
		return nil, fmt.Errorf("leek/get : %w", err)
	}
	s := &LeekSummary{
		ID: raw.ID, Name: raw.Name, Level: raw.Level, Talent: raw.Talent, Farmer: raw.Farmer.ID,
		Stats:      map[string]int{},
		TotalStats: map[string]int{},
		Weapons:    []WeaponRef{},
		Chips:      []ChipRef{},
		Record:     Record{raw.Victories, raw.Draws, raw.Defeats, float64(raw.Ratio)},
		Fights:     []FightLine{},
	}
	for _, name := range statNames {
		s.Stats[name] = intField(flat, name)
		s.TotalStats[name] = intField(flat, "total_"+name)
	}
	for _, w := range raw.Weapons {
		ref := WeaponRef{Item: w.Template, Name: fmt.Sprintf("weapon_item_%d", w.Template)}
		if wp := items.WeaponByItem(w.Template); wp != nil {
			ref.ID, ref.Name = wp.ID, wp.Name
		}
		s.Weapons = append(s.Weapons, ref)
	}
	for _, c := range raw.Chips {
		s.Chips = append(s.Chips, ChipRef{ID: c.Template, Name: items.ChipName(c.Template)})
	}
	if raw.AI != nil {
		s.AI = &AIRef{ID: raw.AI.ID, Name: raw.AI.Name, Valid: raw.AI.Valid}
	}
	for i, f := range raw.Fights {
		if i >= 10 {
			break
		}
		s.Fights = append(s.Fights, fightLine(f, raw.ID))
	}
	if privateJSON != nil {
		var priv rawLeek
		if err := json.Unmarshal(privateJSON, &priv); err != nil {
			return nil, fmt.Errorf("leek/get-private : %w", err)
		}
		s.Capital = priv.Capital
		var privFlat map[string]json.RawMessage
		if err := json.Unmarshal(privateJSON, &privFlat); err != nil {
			return nil, fmt.Errorf("leek/get-private : %w", err)
		}
		stats := map[string]int{}
		for _, name := range statNames {
			stats[name] = intField(privFlat, name)
		}
		s.Spent = CapitalSpent(priv.Level, stats)
		for _, c := range priv.Components {
			if c != nil {
				s.Components = append(s.Components, *c)
			}
		}
	}
	return s, nil
}

func intField(m map[string]json.RawMessage, key string) int {
	var v float64
	if r, ok := m[key]; ok {
		_ = json.Unmarshal(r, &v)
	}
	return int(v)
}

func isoDate(epoch int64) string {
	return time.Unix(epoch, 0).UTC().Format(time.RFC3339)
}
