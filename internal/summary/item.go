package summary

import "github.com/ixigrek/leeks-mcp/internal/leekwars"

// ItemSummary décrit une arme ou une puce de façon uniforme.
type ItemSummary struct {
	Kind       string            `json:"kind"`
	ID         int               `json:"id"`
	Item       int               `json:"item,omitempty"`
	Name       string            `json:"name"`
	Level      int               `json:"level"`
	Cost       int               `json:"cost"`
	MinRange   int               `json:"min_range"`
	MaxRange   int               `json:"max_range"`
	LaunchType int               `json:"launch_type"`
	Area       int               `json:"area"`
	Los        bool              `json:"los"`
	Cooldown   *int              `json:"cooldown,omitempty"`
	MaxUses    int               `json:"max_uses"`
	Effects    []leekwars.Effect `json:"effects"`
}

// Item convertit un résultat de recherche en résumé.
func Item(m leekwars.Match) ItemSummary {
	if m.Weapon != nil {
		w := m.Weapon
		return ItemSummary{
			Kind: "weapon", ID: w.ID, Item: w.Item, Name: w.Name, Level: w.Level,
			Cost: w.Cost, MinRange: w.MinRange, MaxRange: w.MaxRange, LaunchType: w.LaunchType,
			Area: w.Area, Los: w.Los, MaxUses: w.MaxUses, Effects: w.Effects,
		}
	}
	c := m.Chip
	cd := c.Cooldown
	return ItemSummary{
		Kind: "chip", ID: c.ID, Name: c.Name, Level: c.Level,
		Cost: c.Cost, MinRange: c.MinRange, MaxRange: c.MaxRange, LaunchType: c.LaunchType,
		Area: c.Area, Los: c.Los, Cooldown: &cd, MaxUses: c.MaxUses, Effects: c.Effects,
	}
}
