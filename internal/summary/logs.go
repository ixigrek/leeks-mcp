package summary

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// LogEntry est une ligne debug() d'un script.
type LogEntry struct {
	Entity int    `json:"entity"`
	LeekID int    `json:"leek_id,omitempty"`
	AI     string `json:"ai"`
	Line   int    `json:"line"`
	Text   string `json:"text"`
}

// LogTurn regroupe les lignes d'un tour.
type LogTurn struct {
	Turn    int        `json:"turn"`
	Entries []LogEntry `json:"entries"`
}

// LogsSummary est le résumé de fight/get-logs.
type LogsSummary struct {
	Fight int       `json:"fight"`
	Turns []LogTurn `json:"turns"`
}

// Logs regroupe les lignes de fight/get-logs par tour, en recoupant l'index
// d'action avec le rapport fight/get. leekID (0 = tous) filtre sur le poireau.
func Logs(logsJSON, fightJSON []byte, leekID int) (*LogsSummary, error) {
	raw, err := parseFightReport(fightJSON)
	if err != nil {
		return nil, err
	}
	byFarmer, err := phpObject(logsJSON)
	if err != nil {
		return nil, fmt.Errorf("fight/get-logs : %w", err)
	}

	// Tour et entité courante à chaque index d'action.
	n := len(raw.Data.Actions)
	turnAt := make([]int, n)
	casterAt := make([]int, n)
	turn, caster := 1, -1
	for i, a := range raw.Data.Actions {
		switch argInt(a, 0) {
		case actNewTurn:
			turn = argInt(a, 1)
		case actLeekTurn:
			caster = argInt(a, 1)
		}
		turnAt[i], casterAt[i] = turn, caster
	}
	entityLeek := map[int]int{}
	names := raw.leekIDByName()
	for _, l := range raw.Data.Leeks {
		entityLeek[l.ID] = names[l.Name]
	}

	type indexed struct {
		idx   int
		lines [][]json.RawMessage
	}
	var all []indexed
	for _, farmerJSON := range byFarmer {
		byIndex, err := phpObject(farmerJSON)
		if err != nil {
			return nil, fmt.Errorf("fight/get-logs : %w", err)
		}
		for key, linesJSON := range byIndex {
			idx, err := strconv.Atoi(key)
			if err != nil {
				continue
			}
			var lines [][]json.RawMessage
			if err := json.Unmarshal(linesJSON, &lines); err != nil {
				return nil, fmt.Errorf("fight/get-logs : %w", err)
			}
			all = append(all, indexed{idx, lines})
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].idx < all[j].idx })

	s := &LogsSummary{Fight: raw.ID, Turns: []LogTurn{}}
	for _, it := range all {
		t, entity := 1, -1
		if it.idx >= 0 && it.idx < n {
			t, entity = turnAt[it.idx], casterAt[it.idx]
		}
		lid := entityLeek[entity]
		if leekID != 0 && lid != leekID {
			continue
		}
		for _, line := range it.lines {
			entry := LogEntry{Entity: entity, LeekID: lid, Line: argInt(line, 5)}
			if len(line) > 2 {
				_ = json.Unmarshal(line[2], &entry.Text)
			}
			if len(line) > 4 {
				_ = json.Unmarshal(line[4], &entry.AI)
			}
			if len(s.Turns) == 0 || s.Turns[len(s.Turns)-1].Turn != t {
				s.Turns = append(s.Turns, LogTurn{Turn: t})
			}
			last := &s.Turns[len(s.Turns)-1]
			last.Entries = append(last.Entries, entry)
		}
	}
	return s, nil
}

// KeepTurns ne garde que les tours from à to inclus (0 = sans borne).
func (s *LogsSummary) KeepTurns(from, to int) {
	kept := []LogTurn{}
	for _, t := range s.Turns {
		if inTurns(t.Turn, from, to) {
			kept = append(kept, t)
		}
	}
	s.Turns = kept
}

// phpObject lit un objet JSON indexé par clés ; l'API (PHP) renvoie un objet vide
// sous la forme [] (combat sans logs, éleveur sans ligne).
func phpObject(b []byte) (map[string]json.RawMessage, error) {
	var arr []json.RawMessage
	if json.Unmarshal(b, &arr) == nil && len(arr) == 0 {
		return map[string]json.RawMessage{}, nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}
