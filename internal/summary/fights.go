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
	// Noms des adversaires, quand l'API les fournit (forme objet de leeks1/leeks2).
	OpponentNames []string `json:"opponent_names,omitempty"`
	// Nom du boss affronté (combats de boss seulement).
	BossName string `json:"boss_name,omitempty"`
}

// fightLeekRef est un participant d'un combat (fights[].leeks1/leeks2 de leek/get et
// de history/get-leek-history).
// L'API a renvoyé de simples ids puis, depuis octobre 2026, des objets {id, name} ;
// les deux formes sont acceptées.
type fightLeekRef struct {
	ID   int
	Name string
}

func (r *fightLeekRef) UnmarshalJSON(b []byte) error {
	var id int
	if err := json.Unmarshal(b, &id); err == nil {
		*r = fightLeekRef{ID: id}
		return nil
	}
	var obj struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return fmt.Errorf("participant : attendu un id ou un objet {id, name}, reçu %s", b)
	}
	*r = fightLeekRef{ID: obj.ID, Name: obj.Name}
	return nil
}

type rawFight struct {
	ID       int            `json:"id"`
	Date     int64          `json:"date"`
	Type     int            `json:"type"`
	Context  int            `json:"context"`
	Result   string         `json:"result"`
	Duration int            `json:"duration"`
	Leeks1   []fightLeekRef `json:"leeks1"`
	Leeks2   []fightLeekRef `json:"leeks2"`
	BossName string         `json:"boss_name"`
}

func fightLine(f rawFight, leekID int) FightLine {
	opponents := f.Leeks2
	for _, l := range f.Leeks2 {
		if l.ID == leekID {
			opponents = f.Leeks1
			break
		}
	}
	ids := []int{}
	var names []string
	for _, l := range opponents {
		ids = append(ids, l.ID)
		if l.Name != "" {
			names = append(names, l.Name)
		}
	}
	return FightLine{
		ID: f.ID, Date: isoDate(f.Date), Type: f.Type, Context: f.Context,
		Result: f.Result, Duration: f.Duration, Opponents: ids, OpponentNames: names,
		BossName: f.BossName,
	}
}

// FightList extrait l'historique de history/get-leek-history (complet, du plus
// récent au plus ancien ; leek/get n'en donne qu'une douzaine), filtré par résultat
// (win, defeat, draw ou vide) et limité (0 = 10).
func FightList(historyJSON []byte, leekID int, result string, limit int) ([]FightLine, error) {
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
	if err := json.Unmarshal(historyJSON, &raw); err != nil {
		return nil, fmt.Errorf("history/get-leek-history : %w", err)
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
