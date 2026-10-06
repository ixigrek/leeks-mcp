package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// batchRoutes servent trois combats terminés : victoire, défaite (solo), nul.
var batchRoutes = map[string][]string{
	"GET /api/garden/get":                       {"garden.json"},
	"GET /api/garden/get-leek-opponents/135146": {"garden_leek_opponents_135146.json"},
	"GET /api/garden/get-farmer-opponents":      {"garden_farmer_opponents.json"},
	"POST /api/garden/start-solo-fight":         {"start_fight_53996480.json", "start_fight.json", "start_fight_53996476.json"},
	"POST /api/garden/start-farmer-fight":       {"start_fight.json"},
	"POST /api/garden/start-boss-fight":         {"start_fight_53996480.json"},
	"GET /api/fight/get/53996480":               {"fight_53996480.json"},
	"GET /api/fight/get/53994496":               {"fight_pending.json", "fight_53994496.json"},
	"GET /api/fight/get/53996476":               {"fight_53996476.json"},
	"GET /api/fight/get-logs/53996480":          {"logs_53996480.json"},
	"GET /api/fight/get-logs/53994496":          {"logs_53994496.json"},
	"GET /api/fight/get-logs/53996476":          {"logs_53996476.json"},
}

func batchRoutesWith(extra map[string][]string) map[string][]string {
	routes := map[string][]string{}
	for k, v := range batchRoutes {
		routes[k] = v
	}
	for k, v := range extra {
		routes[k] = v
	}
	return routes
}

func decodeBatch(t *testing.T, text string) batchResult {
	t.Helper()
	var out batchResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("%v : %.300s", err, text)
	}
	return out
}

func TestRunBatchSolo(t *testing.T) {
	cs, api := sessionAPI(t, "tok", batchRoutes)
	text, isErr := call(t, cs, "run_batch", map[string]any{"leek_id": 135146, "n": 3})
	if isErr {
		t.Fatal(text)
	}
	out := decodeBatch(t, text)
	if out.Requested != 3 || out.Launched != 3 || out.Fights != 3 || len(out.Errors) != 0 {
		t.Fatalf("lot : %s", text)
	}
	if out.Wins != 1 || out.Draws != 1 || out.Defeats != 1 || len(out.DefeatIDs) != 1 || out.DefeatIDs[0] != 53994496 {
		t.Fatalf("bilan : %s", text)
	}
	if out.Versions["N-2026-10-06d + G00-2026-10-06h"] != 2 || out.Versions["2026-10-06d"] != 1 {
		t.Fatalf("VERSION : %v", out.Versions)
	}
	if out.AvgTurns != 31 || out.AvgLife != 1106 || out.AvgLifeWins != 1587 {
		t.Fatalf("moyennes : %s", text)
	}
	if n := api.count("GET /api/garden/get-leek-opponents/135146"); n != 3 {
		t.Fatalf("adversaires lus %d fois, un tirage par combat attendu", n)
	}
	for _, w := range api.writes() {
		if w.Path != "/api/garden/start-solo-fight" || w.Body["leek_id"] != float64(135146) || w.Body["target_id"] == nil {
			t.Fatalf("POST = %+v", w)
		}
	}
	if n := api.count("GET /api/fight/get/53994496"); n < 2 {
		t.Fatalf("combat en génération sondé %d fois", n)
	}
}

func TestRunBatchBoss(t *testing.T) {
	cs, api := sessionAPI(t, "tok", batchRoutes)
	text, isErr := call(t, cs, "run_batch", map[string]any{"leek_id": 135146, "n": 2, "type": "boss", "boss": "nasu_samurai"})
	if isErr {
		t.Fatal(text)
	}
	if out := decodeBatch(t, text); out.Wins != 2 || out.Launched != 2 {
		t.Fatalf("lot boss : %s", text)
	}
	if n := api.count("GET /api/farmer/get-from-token"); n != 1 {
		t.Fatalf("participants lus %d fois, une seule attendue", n)
	}
	for _, w := range api.writes() {
		if w.Body["boss_id"] != float64(1) || len(w.Body["participants"].([]any)) != 4 {
			t.Fatalf("POST = %+v", w)
		}
	}
}

func TestRunBatchFarmerLeekAbsentIsReported(t *testing.T) {
	cs, _ := sessionAPI(t, "tok", batchRoutes)
	text, isErr := call(t, cs, "run_batch", map[string]any{"leek_id": 1, "n": 1, "type": "farmer"})
	if isErr {
		t.Fatal(text)
	}
	out := decodeBatch(t, text)
	if out.Launched != 1 || out.Fights != 0 || len(out.Errors) != 1 || !strings.Contains(out.Errors[0], "ne participe pas") {
		t.Fatalf("erreur par combat attendue : %s", text)
	}
}

func TestRunBatchStopsOnLaunchError(t *testing.T) {
	cs, api := sessionAPI(t, "tok", batchRoutesWith(map[string][]string{
		"POST /api/garden/start-solo-fight": {"start_fight_53996480.json", "404:start_fight_error.json"},
	}))
	text, isErr := call(t, cs, "run_batch", map[string]any{"leek_id": 135146, "n": 5})
	if isErr {
		t.Fatal(text)
	}
	out := decodeBatch(t, text)
	if out.Launched != 1 || out.Wins != 1 || len(out.Errors) != 1 || !strings.Contains(out.Errors[0], "lancement 2/5") {
		t.Fatalf("arrêt au premier échec attendu : %s", text)
	}
	if n := len(api.writes()); n != 2 {
		t.Fatalf("%d POST, 2 attendus", n)
	}
}

func TestRunBatchRefusals(t *testing.T) {
	cases := []struct {
		args map[string]any
		want string
	}{
		{map[string]any{"leek_id": 135146, "n": 0}, "entre 1 et 50"},
		{map[string]any{"leek_id": 135146, "n": 51}, "entre 1 et 50"},
		{map[string]any{"leek_id": 135146, "n": 1, "type": "team"}, "inconnu"},
		{map[string]any{"leek_id": 135146, "n": 1, "type": "boss"}, "préciser boss"},
		{map[string]any{"leek_id": 135146, "n": 3}, "1 combat(s) restant(s)"},
	}
	for _, c := range cases {
		cs, api := sessionAPI(t, "tok", batchRoutesWith(map[string][]string{
			"GET /api/garden/get": {"garden_one_fight.json"},
		}))
		text, isErr := call(t, cs, "run_batch", c.args)
		if !isErr || !strings.Contains(text, c.want) {
			t.Errorf("%v : %s", c.args, text)
		}
		if n := len(api.writes()); n != 0 {
			t.Errorf("%v : %d POST malgré le refus", c.args, n)
		}
	}
	cs := session(t, "")
	if text, isErr := call(t, cs, "run_batch", map[string]any{"leek_id": 135146, "n": 1}); !isErr || !strings.Contains(text, "token") {
		t.Fatalf("token requis : %s", text)
	}
}
