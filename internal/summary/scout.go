package summary

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
)

// effectKind nomme un type d'effet (Effect.ID) et la stat du lanceur qui le
// multiplie par (1 + stat/100) ; stat vide = valeur brute.
type effectKind struct {
	Name string
	Stat string
}

// effectKinds suit l'enum EffectType du client LeekWars (src/model/effect.ts).
var effectKinds = map[int]effectKind{
	1:  {"damage", "strength"},
	2:  {"heal", "wisdom"},
	3:  {"buff_strength", "science"},
	4:  {"buff_agility", "science"},
	5:  {"relative_shield", "resistance"},
	6:  {"absolute_shield", "resistance"},
	7:  {"buff_mp", "science"},
	8:  {"buff_tp", "science"},
	9:  {"debuff", ""},
	10: {"teleport", ""},
	11: {"invert", ""},
	12: {"boost_max_life", "wisdom"},
	13: {"poison", "magic"},
	14: {"summon", ""},
	15: {"resurrect", ""},
	16: {"kill", ""},
	17: {"shackle_mp", "magic"},
	18: {"shackle_tp", "magic"},
	19: {"shackle_strength", "magic"},
	20: {"damage_return", "agility"},
	21: {"buff_resistance", "science"},
	22: {"buff_wisdom", "science"},
	23: {"antidote", ""},
	24: {"shackle_magic", "magic"},
	25: {"aftereffect", "science"},
	26: {"vulnerability", "magic"},
	27: {"absolute_vulnerability", "magic"},
	28: {"life_damage", ""},
	29: {"steal_absolute_shield", ""},
	30: {"nova_damage", "science"},
	31: {"raw_buff_mp", ""},
	32: {"raw_buff_tp", ""},
	33: {"poison_to_science", ""},
	34: {"damage_to_absolute_shield", ""},
	35: {"damage_to_strength", ""},
	36: {"nova_damage_to_magic", ""},
	37: {"raw_absolute_shield", ""},
	38: {"raw_buff_strength", ""},
	39: {"raw_buff_magic", ""},
	40: {"raw_buff_science", ""},
	41: {"raw_buff_agility", ""},
	42: {"raw_buff_resistance", ""},
	43: {"propagation", ""},
	44: {"raw_buff_wisdom", ""},
	45: {"nova_vitality", ""},
	46: {"attract", ""},
	47: {"shackle_agility", "magic"},
	48: {"shackle_wisdom", "magic"},
	49: {"remove_shackle", ""},
	50: {"moved_to_mp", ""},
	51: {"push", ""},
	52: {"raw_buff_power", ""},
	53: {"repel", ""},
	54: {"raw_relative_shield", ""},
	55: {"ally_killed_to_agility", ""},
	56: {"kill_to_tp", ""},
	57: {"raw_heal", ""},
	58: {"critical_to_heal", ""},
	59: {"add_state", ""},
	60: {"total_debuff", ""},
	61: {"steal_life", ""},
	62: {"multiply_stats", ""},
	63: {"damage_to_resistance", ""},
	64: {"superinfection", ""},
}

// Chance de coup critique : agilité/10 %, dégâts et soins multipliés par criticalFactor.
const criticalFactor = 1.3

// ScoutEffect est un effet d'arme ou de puce chiffré avec les stats du lanceur.
type ScoutEffect struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Stat  string  `json:"stat,omitempty"`
	Min   int     `json:"min"`
	Max   int     `json:"max"`
	Avg   float64 `json:"avg"`
	Turns int     `json:"turns,omitempty"`
}

// ScoutItem est une arme ou une puce équipée, avec ses effets chiffrés.
type ScoutItem struct {
	Name     string        `json:"name"`
	Cost     int           `json:"cost"`
	MinRange int           `json:"min_range"`
	MaxRange int           `json:"max_range"`
	Area     int           `json:"area"`
	Los      bool          `json:"los"`
	Cooldown *int          `json:"cooldown,omitempty"`
	MaxUses  int           `json:"max_uses"`
	Effects  []ScoutEffect `json:"effects"`
}

// Critical décrit les coups critiques du poireau (non appliqués à min/max).
type Critical struct {
	ChancePct float64 `json:"chance_pct"`
	Factor    float64 `json:"factor"`
}

// VsUsFight est un combat de l'adversaire contre nous, résultat de notre point de vue.
type VsUsFight struct {
	ID       int    `json:"id"`
	Date     string `json:"date"`
	Type     int    `json:"type"`
	Context  int    `json:"context"`
	Result   string `json:"result"`
	OurLeeks []int  `json:"our_leeks"`
}

// LeekRecord est le bilan d'un de nos poireaux contre l'adversaire.
type LeekRecord struct {
	LeekID    int `json:"leek_id"`
	Victories int `json:"victories"`
	Draws     int `json:"draws"`
	Defeats   int `json:"defeats"`
}

// VsUs est l'historique de l'adversaire contre nous, de notre point de vue.
type VsUs struct {
	Victories int          `json:"victories"`
	Draws     int          `json:"draws"`
	Defeats   int          `json:"defeats"`
	ByLeek    []LeekRecord `json:"by_leek"`
	Fights    []VsUsFight  `json:"fights"`
}

// ScoutSummary est la fiche d'un adversaire avant un tournoi.
type ScoutSummary struct {
	ID       int            `json:"id"`
	Name     string         `json:"name"`
	Level    int            `json:"level"`
	Talent   int            `json:"talent"`
	Farmer   int            `json:"farmer"`
	AI       *AIRef         `json:"ai"`
	Life     int            `json:"life"`
	Stats    map[string]int `json:"stats"`
	Record   Record         `json:"record"`
	Critical Critical       `json:"critical"`
	Weapons  []ScoutItem    `json:"weapons"`
	Chips    []ScoutItem    `json:"chips"`
	VsUs     *VsUs          `json:"vs_us,omitempty"`
	// Pourquoi vs_us manque (pas de token, historique indisponible).
	Notes []string `json:"notes,omitempty"`
}

// Scout résume leek/get d'un adversaire : stats totales (composants compris),
// équipement avec effets chiffrés selon ses stats.
func Scout(publicJSON []byte, items *leekwars.Items) (*ScoutSummary, error) {
	l, err := Leek(publicJSON, nil, items)
	if err != nil {
		return nil, err
	}
	stats := map[string]int{}
	for k, v := range l.TotalStats {
		if k != "life" {
			stats[k] = v
		}
	}
	s := &ScoutSummary{
		ID: l.ID, Name: l.Name, Level: l.Level, Talent: l.Talent, Farmer: l.Farmer, AI: l.AI,
		Life: l.TotalStats["life"], Stats: stats, Record: l.Record,
		Critical: Critical{ChancePct: float64(stats["agility"]) / 10, Factor: criticalFactor},
		Weapons:  []ScoutItem{},
		Chips:    []ScoutItem{},
	}
	for _, w := range l.Weapons {
		wp := items.WeaponByItem(w.Item)
		if wp == nil {
			s.Weapons = append(s.Weapons, ScoutItem{Name: w.Name, Effects: []ScoutEffect{}})
			continue
		}
		s.Weapons = append(s.Weapons, ScoutItem{
			Name: wp.Name, Cost: wp.Cost, MinRange: wp.MinRange, MaxRange: wp.MaxRange,
			Area: wp.Area, Los: wp.Los, MaxUses: wp.MaxUses, Effects: scoutEffects(wp.Effects, stats),
		})
	}
	for _, c := range l.Chips {
		ch := items.ChipByID(c.ID)
		if ch == nil {
			s.Chips = append(s.Chips, ScoutItem{Name: c.Name, Effects: []ScoutEffect{}})
			continue
		}
		cd := ch.Cooldown
		s.Chips = append(s.Chips, ScoutItem{
			Name: ch.Name, Cost: ch.Cost, MinRange: ch.MinRange, MaxRange: ch.MaxRange,
			Area: ch.Area, Los: ch.Los, Cooldown: &cd, MaxUses: ch.MaxUses, Effects: scoutEffects(ch.Effects, stats),
		})
	}
	return s, nil
}

// scoutEffects chiffre des effets : value1 à value1 + value2, multipliés par
// (1 + stat/100) pour les effets qui suivent une stat.
func scoutEffects(effects []leekwars.Effect, stats map[string]int) []ScoutEffect {
	out := []ScoutEffect{}
	for _, e := range effects {
		k, ok := effectKinds[e.ID]
		if !ok {
			k = effectKind{Name: fmt.Sprintf("effect_%d", e.ID)}
		}
		mult := 1.0
		if k.Stat != "" {
			mult += float64(stats[k.Stat]) / 100
		}
		lo, hi := e.Value1*mult, (e.Value1+e.Value2)*mult
		out = append(out, ScoutEffect{
			ID: e.ID, Name: k.Name, Stat: k.Stat,
			Min: int(math.Round(lo)), Max: int(math.Round(hi)),
			Avg: math.Round((lo+hi)/2*10) / 10, Turns: e.Turns,
		})
	}
	return out
}

// ScoutVsUs extrait de history/get-leek-history de l'adversaire leekID les
// combats contre nous : un de nos poireaux en face, ou notre éleveur en face
// (combats d'éleveur). Résultats de notre point de vue ; fights limité (0 = 10).
func ScoutVsUs(historyJSON []byte, leekID, ourFarmer int, ourLeeks []int, limit int) (*VsUs, error) {
	if limit <= 0 {
		limit = 10
	}
	var raw struct {
		Fights []struct {
			rawFight
			Farmer1 int `json:"farmer1"`
			Farmer2 int `json:"farmer2"`
		} `json:"fights"`
	}
	if err := json.Unmarshal(historyJSON, &raw); err != nil {
		return nil, fmt.Errorf("history/get-leek-history : %w", err)
	}
	ours := map[int]bool{}
	for _, id := range ourLeeks {
		ours[id] = true
	}
	v := &VsUs{ByLeek: []LeekRecord{}, Fights: []VsUsFight{}}
	byLeek := map[int]*LeekRecord{}
	for _, f := range raw.Fights {
		side, farmer := f.Leeks2, f.Farmer2
		for _, l := range f.Leeks2 {
			if l.ID == leekID {
				side, farmer = f.Leeks1, f.Farmer1
				break
			}
		}
		var engaged []int
		for _, l := range side {
			if ours[l.ID] {
				engaged = append(engaged, l.ID)
			}
		}
		if len(engaged) == 0 && (ourFarmer <= 0 || farmer != ourFarmer) {
			continue
		}
		if engaged == nil {
			// Combat d'éleveur dont nos poireaux ont changé depuis : tout le camp.
			for _, l := range side {
				engaged = append(engaged, l.ID)
			}
		}
		result := map[string]string{"win": "defeat", "defeat": "win"}[f.Result]
		if result == "" {
			result = f.Result
		}
		for _, id := range engaged {
			r := byLeek[id]
			if r == nil {
				r = &LeekRecord{LeekID: id}
				byLeek[id] = r
			}
			tally(&r.Victories, &r.Draws, &r.Defeats, result)
		}
		tally(&v.Victories, &v.Draws, &v.Defeats, result)
		if len(v.Fights) < limit {
			v.Fights = append(v.Fights, VsUsFight{
				ID: f.ID, Date: isoDate(f.Date), Type: f.Type, Context: f.Context,
				Result: result, OurLeeks: engaged,
			})
		}
	}
	for _, r := range byLeek {
		v.ByLeek = append(v.ByLeek, *r)
	}
	sort.Slice(v.ByLeek, func(i, j int) bool { return v.ByLeek[i].LeekID < v.ByLeek[j].LeekID })
	return v, nil
}

func tally(win, draw, defeat *int, result string) {
	switch result {
	case "win":
		*win++
	case "draw":
		*draw++
	case "defeat":
		*defeat++
	}
}
