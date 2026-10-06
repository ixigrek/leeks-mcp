package summary

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
)

// LeekRef est un poireau d'un éleveur.
type LeekRef struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// Stock est un objet possédé non équipé.
type Stock struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Inventory regroupe les objets non équipés (données privées).
type Inventory struct {
	Weapons []Stock `json:"weapons"`
	Chips   []Stock `json:"chips"`
}

// FarmerSummary résume farmer/get ou farmer/get-from-token.
type FarmerSummary struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Talent    int       `json:"talent"`
	Record    Record    `json:"record"`
	Habs      *int      `json:"habs,omitempty"`
	Crystals  *int      `json:"crystals,omitempty"`
	Leeks     []LeekRef `json:"leeks"`
	Inventory Inventory `json:"inventory"`
}

type rawStock struct {
	ID       int `json:"id"`
	Template int `json:"template"`
	Quantity int `json:"quantity"`
}

// Farmer résume une réponse farmer/* (enveloppe {"farmer": {...}}).
func Farmer(body []byte, items *leekwars.Items) (*FarmerSummary, error) {
	var env struct {
		Farmer struct {
			ID        int                `json:"id"`
			Name      string             `json:"name"`
			Talent    int                `json:"talent"`
			Victories int                `json:"victories"`
			Draws     int                `json:"draws"`
			Defeats   int                `json:"defeats"`
			Ratio     flexFloat          `json:"ratio"`
			Habs      *int               `json:"habs"`
			Crystals  *int               `json:"crystals"`
			Leeks     map[string]LeekRef `json:"leeks"`
			Weapons   []rawStock         `json:"weapons"`
			Chips     []rawStock         `json:"chips"`
		} `json:"farmer"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("farmer : %w", err)
	}
	f := env.Farmer
	s := &FarmerSummary{
		ID: f.ID, Name: f.Name, Talent: f.Talent,
		Record:    Record{f.Victories, f.Draws, f.Defeats, float64(f.Ratio)},
		Habs:      f.Habs,
		Crystals:  f.Crystals,
		Leeks:     []LeekRef{},
		Inventory: Inventory{Weapons: []Stock{}, Chips: []Stock{}},
	}
	for _, l := range f.Leeks {
		s.Leeks = append(s.Leeks, l)
	}
	sort.Slice(s.Leeks, func(i, j int) bool { return s.Leeks[i].ID < s.Leeks[j].ID })
	for _, w := range f.Weapons {
		name := fmt.Sprintf("weapon_item_%d", w.Template)
		if wp := items.WeaponByItem(w.Template); wp != nil {
			name = wp.Name
		}
		s.Inventory.Weapons = append(s.Inventory.Weapons, Stock{ID: w.Template, Name: name, Count: w.Quantity})
	}
	for _, c := range f.Chips {
		s.Inventory.Chips = append(s.Inventory.Chips, Stock{ID: c.Template, Name: items.ChipName(c.Template), Count: c.Quantity})
	}
	return s, nil
}
