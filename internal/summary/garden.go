package summary

import (
	"encoding/json"
	"fmt"
	"sort"
)

// FightFinished est le status d'un rapport fight/get dont la génération est terminée.
const FightFinished = 2

// FightStatus lit le status d'un rapport fight/get (< FightFinished = en génération).
func FightStatus(body []byte) (int, error) {
	var raw struct {
		Status *int `json:"status"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, fmt.Errorf("fight/get : %w", err)
	}
	if raw.Status == nil {
		return 0, fmt.Errorf("fight/get : champ status absent")
	}
	return *raw.Status, nil
}

// StartedFightID extrait l'id du combat créé par garden/start-*-fight. Le catalogue
// de l'API annonce fight_id, le client officiel lit fight, les lots renvoient fights.
func StartedFightID(body []byte) (int, error) {
	var raw struct {
		Fight   int   `json:"fight"`
		FightID int   `json:"fight_id"`
		Fights  []int `json:"fights"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, fmt.Errorf("réponse de lancement : %w", err)
	}
	switch {
	case raw.Fight > 0:
		return raw.Fight, nil
	case raw.FightID > 0:
		return raw.FightID, nil
	case len(raw.Fights) > 0 && raw.Fights[0] > 0:
		return raw.Fights[0], nil
	}
	return 0, fmt.Errorf("réponse de lancement sans id de combat : %.200s", body)
}

// FarmerLeekIDs liste, triés, les ids des poireaux de farmer/get-from-token.
func FarmerLeekIDs(body []byte) ([]int, error) {
	var raw struct {
		Farmer struct {
			Leeks map[string]struct {
				ID int `json:"id"`
			} `json:"leeks"`
		} `json:"farmer"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("farmer/get-from-token : %w", err)
	}
	ids := make([]int, 0, len(raw.Farmer.Leeks))
	for _, l := range raw.Farmer.Leeks {
		ids = append(ids, l.ID)
	}
	sort.Ints(ids)
	return ids, nil
}

// BossRef identifie un boss de boss/get-all.
type BossRef struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// Bosses lit la liste des boss de boss/get-all.
func Bosses(body []byte) ([]BossRef, error) {
	var raw struct {
		Bosses []BossRef `json:"bosses"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("boss/get-all : %w", err)
	}
	return raw.Bosses, nil
}

// Ref identifie un éleveur ou une équipe rattaché à un adversaire.
type Ref struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// CompositionRef décrit une composition d'équipe du potager.
type CompositionRef struct {
	ID     int       `json:"id"`
	Name   string    `json:"name"`
	Fights int       `json:"fights"`
	Leeks  []LeekRef `json:"leeks"`
}

// GardenSummary résume garden/get : combats restants et compositions.
type GardenSummary struct {
	Fights              int              `json:"fights"`
	MaxFights           int              `json:"max_fights"`
	TeamFights          int              `json:"team_fights"`
	MaxTeamFights       int              `json:"max_team_fights"`
	FarmerEnabled       bool             `json:"farmer_enabled"`
	TeamEnabled         bool             `json:"team_enabled"`
	BattleRoyaleEnabled bool             `json:"battle_royale_enabled"`
	Compositions        []CompositionRef `json:"compositions"`
}

// Garden lit l'état du potager renvoyé par garden/get.
func Garden(body []byte) (*GardenSummary, error) {
	var raw struct {
		Garden struct {
			Fights              int              `json:"fights"`
			MaxFights           int              `json:"max_fights"`
			TeamFights          int              `json:"team_fights"`
			MaxTeamFights       int              `json:"max_team_fights"`
			FarmerEnabled       bool             `json:"farmer_enabled"`
			TeamEnabled         bool             `json:"team_enabled"`
			BattleRoyaleEnabled bool             `json:"battle_royale_enabled"`
			Compositions        []CompositionRef `json:"my_compositions"`
		} `json:"garden"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("garden/get : %w", err)
	}
	g := raw.Garden
	if g.Compositions == nil {
		g.Compositions = []CompositionRef{}
	}
	for i := range g.Compositions {
		if g.Compositions[i].Leeks == nil {
			g.Compositions[i].Leeks = []LeekRef{}
		}
	}
	return &GardenSummary{
		Fights: g.Fights, MaxFights: g.MaxFights, TeamFights: g.TeamFights, MaxTeamFights: g.MaxTeamFights,
		FarmerEnabled: g.FarmerEnabled, TeamEnabled: g.TeamEnabled, BattleRoyaleEnabled: g.BattleRoyaleEnabled,
		Compositions: g.Compositions,
	}, nil
}

// Opponent est un adversaire proposé par le matchmaking : poireau (level, talent),
// éleveur (leek_count, total_level) ou composition (leeks, team). Les champs
// absents de la forme concernée sont omis.
type Opponent struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Level      int       `json:"level,omitempty"`
	Talent     int       `json:"talent,omitempty"`
	LeekCount  int       `json:"leek_count,omitempty"`
	TotalLevel int       `json:"total_level,omitempty"`
	Leeks      []LeekRef `json:"leeks,omitempty"`
	Team       *Ref      `json:"team,omitempty"`
}

// Opponents lit la liste d'adversaires de garden/get-*-opponents.
func Opponents(body []byte) ([]Opponent, error) {
	var raw struct {
		Opponents []Opponent `json:"opponents"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("adversaires : %w", err)
	}
	if raw.Opponents == nil {
		raw.Opponents = []Opponent{}
	}
	return raw.Opponents, nil
}
