package summary

import (
	"encoding/json"
	"fmt"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
)

// LoadoutComponent est un composant d'un loadout : son emplacement (0 à 7), son
// template et, pour une pièce altérée, ses stats (null pour une pièce de base),
// transmises telles quelles à l'API.
type LoadoutComponent struct {
	Index    int             `json:"index"`
	Template int             `json:"template"`
	Stats    json.RawMessage `json:"stats"`
}

// Loadout est un ensemble d'équipement tel que l'API le renvoie : templates
// d'armes (item) et de puces, composants, et capital dépensé par stat.
type Loadout struct {
	ID               int                `json:"id"`
	Name             string             `json:"name"`
	Icon             string             `json:"icon"`
	Weapons          []int              `json:"weapons"`
	ForgottenWeapons []int              `json:"forgotten_weapons"`
	Chips            []int              `json:"chips"`
	Components       []LoadoutComponent `json:"components"`
	Stats            map[string]int     `json:"stats"`
	Order            int                `json:"order"`
}

// LoadoutSummary est un loadout résumé par noms.
type LoadoutSummary struct {
	ID         int            `json:"id"`
	Name       string         `json:"name"`
	Icon       string         `json:"icon,omitempty"`
	Weapons    []string       `json:"weapons"`
	Chips      []string       `json:"chips"`
	Components int            `json:"components"`
	Stats      map[string]int `json:"stats"`
	Capital    int            `json:"capital"`
}

// LoadoutList résume loadout/get-all : les loadouts et les objets possédés.
type LoadoutList struct {
	Loadouts     []LoadoutSummary `json:"loadouts"`
	OwnedWeapons []string         `json:"owned_weapons"`
	OwnedChips   []string         `json:"owned_chips"`
}

// Loadouts lit loadout/get-all.
func Loadouts(body []byte, items *leekwars.Items) (*LoadoutList, error) {
	var raw struct {
		Loadouts     []Loadout `json:"loadouts"`
		OwnedWeapons []int     `json:"owned_weapons"`
		OwnedChips   []int     `json:"owned_chips"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("loadout/get-all : %w", err)
	}
	out := &LoadoutList{Loadouts: []LoadoutSummary{}, OwnedWeapons: weaponNames(raw.OwnedWeapons, items), OwnedChips: chipNames(raw.OwnedChips, items)}
	for _, l := range raw.Loadouts {
		out.Loadouts = append(out.Loadouts, summarizeLoadout(l, items))
	}
	return out, nil
}

// LoadoutSet lit le loadout renvoyé par loadout/create et loadout/update (champ set).
func LoadoutSet(body []byte) (*Loadout, error) {
	var raw struct {
		Set *Loadout `json:"set"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || raw.Set == nil {
		return nil, fmt.Errorf("réponse loadout sans set : %.200s", body)
	}
	return raw.Set, nil
}

// FindLoadout cherche un loadout par id dans loadout/get-all.
func FindLoadout(body []byte, id int) (*Loadout, error) {
	var raw struct {
		Loadouts []Loadout `json:"loadouts"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("loadout/get-all : %w", err)
	}
	for i := range raw.Loadouts {
		if raw.Loadouts[i].ID == id {
			return &raw.Loadouts[i], nil
		}
	}
	return nil, fmt.Errorf("loadout %d introuvable", id)
}

// Summarize résume un loadout par noms.
func (l Loadout) Summarize(items *leekwars.Items) LoadoutSummary { return summarizeLoadout(l, items) }

func summarizeLoadout(l Loadout, items *leekwars.Items) LoadoutSummary {
	s := LoadoutSummary{
		ID: l.ID, Name: l.Name, Icon: l.Icon,
		Weapons: weaponNames(append(append([]int{}, l.Weapons...), l.ForgottenWeapons...), items),
		Chips:   chipNames(l.Chips, items), Components: len(l.Components), Stats: l.Stats,
	}
	if s.Stats == nil {
		s.Stats = map[string]int{}
	}
	for _, c := range s.Stats {
		s.Capital += c
	}
	return s
}

// ApplyResult résume loadout/apply : l'équipement final du poireau et ce qui a changé.
type ApplyResult struct {
	LeekID       int      `json:"leek_id"`
	Weapons      []string `json:"weapons"`
	Chips        []string `json:"chips"`
	Components   int      `json:"components"`
	StatsChanged bool     `json:"stats_changed"`
	RestatUsed   bool     `json:"restat_used"`
	Skipped      []any    `json:"skipped"`
}

// Applied lit la réponse de loadout/apply.
func Applied(body []byte, leekID int, items *leekwars.Items) (*ApplyResult, error) {
	var raw struct {
		Leek struct {
			Weapons []struct {
				Template int `json:"template"`
			} `json:"weapons"`
			Chips []struct {
				Template int `json:"template"`
			} `json:"chips"`
			Components []json.RawMessage `json:"components"`
		} `json:"leek"`
		Skipped      []any `json:"skipped"`
		StatsChanged bool  `json:"stats_changed"`
		RestatUsed   bool  `json:"restat_used"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("loadout/apply : %w", err)
	}
	out := &ApplyResult{LeekID: leekID, Weapons: []string{}, Chips: []string{}, StatsChanged: raw.StatsChanged, RestatUsed: raw.RestatUsed, Skipped: raw.Skipped}
	if out.Skipped == nil {
		out.Skipped = []any{}
	}
	for _, w := range raw.Leek.Weapons {
		out.Weapons = append(out.Weapons, weaponName(w.Template, items))
	}
	for _, c := range raw.Leek.Chips {
		out.Chips = append(out.Chips, items.ChipName(c.Template))
	}
	for _, c := range raw.Leek.Components {
		if string(c) != "null" {
			out.Components++
		}
	}
	return out, nil
}

func weaponName(item int, items *leekwars.Items) string {
	if w := items.WeaponByItem(item); w != nil {
		return w.Name
	}
	return fmt.Sprintf("weapon_item_%d", item)
}

func weaponNames(ids []int, items *leekwars.Items) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, weaponName(id, items))
	}
	return out
}

func chipNames(ids []int, items *leekwars.Items) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, items.ChipName(id))
	}
	return out
}
