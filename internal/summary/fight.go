package summary

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
)

// Codes d'action du rapport (enum ActionType du client LeekWars).
const (
	actStartFight     = 0
	actEndFight       = 4
	actPlayerDead     = 5
	actNewTurn        = 6
	actLeekTurn       = 7
	actEndTurn        = 8
	actSummon         = 9
	actMoveTo         = 10
	actKill           = 11
	actUseChip        = 12
	actSetWeapon      = 13
	actStackEffect    = 14
	actOpenChest      = 15
	actUseWeapon      = 16
	actPlantAwake     = 17
	actPlantAsleep    = 18
	actTPLost         = 100
	actLifeLost       = 101
	actMPLost         = 102
	actCare           = 103
	actBoostVita      = 104
	actResurrection   = 105
	actNovaDamage     = 107
	actDamageReturn   = 108
	actLifeDamage     = 109
	actPoisonDamage   = 110
	actAftereffect    = 111
	actNovaVitality   = 112
	actSayOld         = 200
	actLama           = 201
	actShowOld        = 202
	actSay            = 203
	actShow           = 205
	actAddWeaponEff   = 301
	actAddChipEff     = 302
	actRemoveEffect   = 303
	actUpdateEffect   = 304
	actAddStackedEff  = 305
	actReduceEffects  = 306
	actRemovePoisons  = 307
	actRemoveShackles = 308
	actUpdateEffTurns = 309
	actBug            = 1002
)

// ignoredActions sont connues mais sans effet sur le résumé.
var ignoredActions = map[int]bool{
	actStartFight: true, actEndFight: true, actEndTurn: true, actKill: true,
	actStackEffect: true, actOpenChest: true, actPlantAwake: true, actPlantAsleep: true,
	actTPLost: true, actMPLost: true, actResurrection: true, actNovaDamage: true,
	actSayOld: true, actLama: true, actShowOld: true, actSay: true, actShow: true,
	actAddWeaponEff: true, actAddChipEff: true, actRemoveEffect: true, actUpdateEffect: true,
	actAddStackedEff: true, actReduceEffects: true, actRemovePoisons: true, actRemoveShackles: true,
	actUpdateEffTurns: true, actBug: true,
}

// EntitySummary décrit une entité du combat (poireau, invocation ou coffre).
type EntitySummary struct {
	ID         int    `json:"id"`
	LeekID     int    `json:"leek_id,omitempty"`
	Name       string `json:"name"`
	Level      int    `json:"level"`
	Team       int    `json:"team"`
	Life       int    `json:"life"`
	Strength   int    `json:"strength"`
	Wisdom     int    `json:"wisdom"`
	Agility    int    `json:"agility"`
	Resistance int    `json:"resistance"`
	Science    int    `json:"science"`
	Magic      int    `json:"magic"`
	TP         int    `json:"tp"`
	MP         int    `json:"mp"`
	Summon     bool   `json:"summon,omitempty"`
}

// Move est un déplacement.
type Move struct {
	To int `json:"to"`
	MP int `json:"mp"`
}

// Shot est un tir d'arme.
type Shot struct {
	Weapon   string `json:"weapon"`
	Cell     int    `json:"cell"`
	Critical bool   `json:"critical,omitempty"`
}

// ChipUse est un lancer de puce.
type ChipUse struct {
	Chip     string `json:"chip"`
	Cell     int    `json:"cell"`
	Critical bool   `json:"critical,omitempty"`
}

// EntityTurn est l'activité d'une entité pendant un tour.
type EntityTurn struct {
	ID          int       `json:"id"`
	Moves       []Move    `json:"moves,omitempty"`
	WeaponShots []Shot    `json:"weapon_shots,omitempty"`
	Chips       []ChipUse `json:"chips,omitempty"`
	Summons     []int     `json:"summons,omitempty"`
	DamageDealt int       `json:"damage_dealt,omitempty"`
	DamageTaken int       `json:"damage_taken,omitempty"`
	Heal        int       `json:"heal,omitempty"`
	Vitality    int       `json:"vitality,omitempty"`
	LifeEnd     int       `json:"life_end"`
}

// TurnSummary regroupe les entités actives d'un tour.
type TurnSummary struct {
	Turn     int          `json:"turn"`
	Entities []EntityTurn `json:"entities"`
	index    map[int]int
}

// Death est la mort d'une entité.
type Death struct {
	Turn   int `json:"turn"`
	Entity int `json:"entity"`
	Killer int `json:"killer,omitempty"`
}

// FightSummary est le résumé de fight/get.
type FightSummary struct {
	ID             int             `json:"id"`
	Date           string          `json:"date"`
	Winner         int             `json:"winner"`
	Duration       int             `json:"duration"`
	Entities       []EntitySummary `json:"entities"`
	Turns          []TurnSummary   `json:"turns"`
	Deaths         []Death         `json:"deaths"`
	UnknownActions map[string]int  `json:"unknown_actions"`
}

type rawFightLeek struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type rawFightReport struct {
	ID     int            `json:"id"`
	Date   int64          `json:"date"`
	Winner int            `json:"winner"`
	Leeks1 []rawFightLeek `json:"leeks1"`
	Leeks2 []rawFightLeek `json:"leeks2"`
	Data   struct {
		Leeks []struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			Level      int    `json:"level"`
			Team       int    `json:"team"`
			Life       int    `json:"life"`
			Strength   int    `json:"strength"`
			Wisdom     int    `json:"wisdom"`
			Agility    int    `json:"agility"`
			Resistance int    `json:"resistance"`
			Science    int    `json:"science"`
			Magic      int    `json:"magic"`
			TP         int    `json:"tp"`
			MP         int    `json:"mp"`
			Summon     bool   `json:"summon"`
		} `json:"leeks"`
		Actions [][]json.RawMessage `json:"actions"`
	} `json:"data"`
}

func parseFightReport(body []byte) (*rawFightReport, error) {
	var raw rawFightReport
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("fight/get : %w", err)
	}
	return &raw, nil
}

// leekIDByName relie le nom d'une entité à l'id public du poireau.
func (r *rawFightReport) leekIDByName() map[string]int {
	m := map[string]int{}
	for _, l := range r.Leeks1 {
		m[l.Name] = l.ID
	}
	for _, l := range r.Leeks2 {
		m[l.Name] = l.ID
	}
	return m
}

func argInt(a []json.RawMessage, i int) int {
	if i >= len(a) {
		return 0
	}
	var v float64
	if json.Unmarshal(a[i], &v) != nil {
		return 0
	}
	return int(v)
}

func argLen(a []json.RawMessage, i int) int {
	if i >= len(a) {
		return 0
	}
	var v []json.RawMessage
	if json.Unmarshal(a[i], &v) != nil {
		return 0
	}
	return len(v)
}

// Fight résume un rapport de combat. leekID (0 = tous) ne garde dans les tours
// que l'activité de ce poireau ; les entités et les morts restent complètes.
func Fight(body []byte, items *leekwars.Items, leekID int) (*FightSummary, error) {
	raw, err := parseFightReport(body)
	if err != nil {
		return nil, err
	}
	names := raw.leekIDByName()
	keep := -1
	if leekID != 0 {
		for _, l := range raw.Data.Leeks {
			if names[l.Name] == leekID {
				keep = l.ID
			}
		}
		if keep < 0 {
			return nil, fmt.Errorf("le poireau %d ne participe pas au combat %d", leekID, raw.ID)
		}
	}
	s := &FightSummary{
		ID: raw.ID, Date: isoDate(raw.Date), Winner: raw.Winner,
		Entities:       []EntitySummary{},
		Turns:          []TurnSummary{},
		Deaths:         []Death{},
		UnknownActions: map[string]int{},
	}
	life := map[int]int{}
	for _, l := range raw.Data.Leeks {
		s.Entities = append(s.Entities, EntitySummary{
			ID: l.ID, LeekID: names[l.Name], Name: l.Name, Level: l.Level, Team: l.Team, Life: l.Life,
			Strength: l.Strength, Wisdom: l.Wisdom, Agility: l.Agility, Resistance: l.Resistance,
			Science: l.Science, Magic: l.Magic, TP: l.TP, MP: l.MP, Summon: l.Summon,
		})
		life[l.ID] = l.Life
	}

	turn := &TurnSummary{Turn: 1, index: map[int]int{}}
	caster := -1
	weapon := map[int]int{}
	closeTurn := func() {
		kept := turn.Entities[:0]
		for _, e := range turn.Entities {
			if keep >= 0 && e.ID != keep {
				continue
			}
			e.LifeEnd = life[e.ID]
			kept = append(kept, e)
		}
		turn.Entities = kept
		turn.index = nil
		s.Turns = append(s.Turns, *turn)
	}
	for _, a := range raw.Data.Actions {
		if len(a) == 0 {
			continue
		}
		code := argInt(a, 0)
		switch code {
		case actNewTurn:
			closeTurn()
			turn = &TurnSummary{Turn: argInt(a, 1), index: map[int]int{}}
		case actLeekTurn:
			caster = argInt(a, 1)
			turn.entity(caster)
		case actSummon:
			summoner, summoned := argInt(a, 1), argInt(a, 2)
			e := turn.entity(summoner)
			e.Summons = append(e.Summons, summoned)
		case actMoveTo:
			e := turn.entity(argInt(a, 1))
			e.Moves = append(e.Moves, Move{To: argInt(a, 2), MP: argLen(a, 3)})
		case actUseChip:
			if caster >= 0 {
				e := turn.entity(caster)
				e.Chips = append(e.Chips, ChipUse{Chip: items.ChipNameByTemplate(argInt(a, 1)), Cell: argInt(a, 2), Critical: argInt(a, 3) == 2})
			}
		case actSetWeapon:
			if caster >= 0 {
				weapon[caster] = argInt(a, 1)
			}
		case actUseWeapon:
			if caster >= 0 {
				e := turn.entity(caster)
				e.WeaponShots = append(e.WeaponShots, Shot{Weapon: items.WeaponName(weapon[caster]), Cell: argInt(a, 1), Critical: argInt(a, 2) == 2})
			}
		case actLifeLost, actDamageReturn, actLifeDamage, actPoisonDamage, actAftereffect:
			victim, pv := argInt(a, 1), argInt(a, 2)
			life[victim] -= pv
			turn.entity(victim).DamageTaken += pv
			if caster >= 0 && caster != victim {
				turn.entity(caster).DamageDealt += pv
			}
		case actCare:
			target, pv := argInt(a, 1), argInt(a, 2)
			life[target] += pv
			turn.entity(target).Heal += pv
		case actBoostVita, actNovaVitality:
			target, pv := argInt(a, 1), argInt(a, 2)
			life[target] += pv
			turn.entity(target).Vitality += pv
		case actPlayerDead:
			s.Deaths = append(s.Deaths, Death{Turn: turn.Turn, Entity: argInt(a, 1), Killer: argInt(a, 2)})
		default:
			if !ignoredActions[code] {
				s.UnknownActions[strconv.Itoa(code)]++
			}
		}
	}
	closeTurn()
	s.Duration = len(s.Turns)
	sort.Slice(s.Entities, func(i, j int) bool { return s.Entities[i].ID < s.Entities[j].ID })
	return s, nil
}

// entity renvoie l'activité de l'entité dans ce tour, créée au besoin.
func (t *TurnSummary) entity(id int) *EntityTurn {
	if i, ok := t.index[id]; ok {
		return &t.Entities[i]
	}
	t.Entities = append(t.Entities, EntityTurn{ID: id})
	t.index[id] = len(t.Entities) - 1
	return &t.Entities[len(t.Entities)-1]
}
