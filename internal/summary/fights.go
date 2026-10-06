package summary

import (
	"encoding/json"
	"fmt"
)

// FightLine est une ligne de l'historique de combats d'un poireau.
type FightLine struct {
	ID        int    `json:"id"`
	Date      string `json:"date"`
	Type      int    `json:"type"`
	Context   int    `json:"context"`
	Result    string `json:"result"`
	Duration  int    `json:"duration"`
	Opponents []int  `json:"opponents"`
}

type rawFight struct {
	ID       int    `json:"id"`
	Date     int64  `json:"date"`
	Type     int    `json:"type"`
	Context  int    `json:"context"`
	Result   string `json:"result"`
	Duration int    `json:"duration"`
	Leeks1   []int  `json:"leeks1"`
	Leeks2   []int  `json:"leeks2"`
}

func fightLine(f rawFight, leekID int) FightLine {
	opponents := f.Leeks2
	for _, id := range f.Leeks2 {
		if id == leekID {
			opponents = f.Leeks1
			break
		}
	}
	if opponents == nil {
		opponents = []int{}
	}
	return FightLine{
		ID: f.ID, Date: isoDate(f.Date), Type: f.Type, Context: f.Context,
		Result: f.Result, Duration: f.Duration, Opponents: opponents,
	}
}

// FightList extrait l'historique de leek/get, filtré par résultat (win, defeat,
// draw ou vide) et limité (0 = 10).
func FightList(leekJSON []byte, leekID int, result string, limit int) ([]FightLine, error) {
	switch result {
	case "", "win", "defeat", "draw":
	default:
		return nil, fmt.Errorf("result %q inconnu : attendu win, defeat ou draw", result)
	}
	if limit <= 0 {
		limit = 10
	}
	var raw struct {
		Fights []rawFight `json:"fights"`
	}
	if err := json.Unmarshal(leekJSON, &raw); err != nil {
		return nil, fmt.Errorf("leek/get : %w", err)
	}
	out := []FightLine{}
	for _, f := range raw.Fights {
		if result != "" && f.Result != result {
			continue
		}
		out = append(out, fightLine(f, leekID))
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
