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
