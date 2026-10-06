package main

import (
	"strings"
	"testing"
)

var gardenRoutes = map[string][]string{
	"GET /api/garden/get":                         {"garden.json"},
	"GET /api/garden/get-leek-opponents/135146":   {"garden_leek_opponents_135146.json"},
	"GET /api/garden/get-farmer-opponents":        {"garden_farmer_opponents.json"},
	"GET /api/garden/get-composition-opponents/7": {"garden_composition_opponents.json"},
	"POST /api/garden/start-solo-fight":           {"start_fight.json"},
	"POST /api/garden/start-farmer-fight":         {"start_fight.json"},
	"POST /api/garden/start-team-fight":           {"start_fight.json"},
	"POST /api/garden/start-boss-fight":           {"start_boss_fight.json"},
	"GET /api/fight/get/53994496":                 {"fight_pending.json", "fight_53994496.json"},
	"GET /api/fight/get/53994497":                 {"fight_53994497.json"},
}

func TestGetGardenSummaryAndOpponents(t *testing.T) {
	cs, _ := sessionAPI(t, "tok", gardenRoutes)
	text, isErr := call(t, cs, "get_garden", map[string]any{})
	if isErr || !strings.Contains(text, `"fights":3040`) || strings.Contains(text, "opponents") {
		t.Fatalf("potager : %.300s", text)
	}
	text, isErr = call(t, cs, "get_garden", map[string]any{"leek_id": 135146})
	if isErr || !strings.Contains(text, `"opponents_for":"leek 135146"`) || !strings.Contains(text, "Imarmaleek") {
		t.Fatalf("adversaires solo : %.300s", text)
	}
	text, isErr = call(t, cs, "get_garden", map[string]any{"farmer": true})
	if isErr || !strings.Contains(text, `"opponents_for":"farmer"`) || !strings.Contains(text, "Simon200079") {
		t.Fatalf("adversaires éleveur : %.300s", text)
	}
	text, isErr = call(t, cs, "get_garden", map[string]any{"composition_id": 7})
	if isErr || !strings.Contains(text, `"opponents_for":"composition 7"`) || !strings.Contains(text, "Potager Uni") {
		t.Fatalf("adversaires équipe : %.300s", text)
	}
	text, isErr = call(t, cs, "get_garden", map[string]any{"leek_id": 135146, "farmer": true})
	if !isErr {
		t.Fatalf("un seul sélecteur attendu : %.300s", text)
	}
	raw, isErr := call(t, cs, "get_garden", map[string]any{"raw": true})
	if isErr || !strings.Contains(raw, `"max_solo_fights"`) {
		t.Fatalf("brut : %.300s", raw)
	}
}

func TestGetGardenRequiresToken(t *testing.T) {
	cs, _ := sessionAPI(t, "", gardenRoutes)
	text, isErr := call(t, cs, "get_garden", map[string]any{})
	if !isErr || !strings.Contains(text, "LEEKWARS_TOKEN") {
		t.Fatalf("erreur de token attendue : %s", text)
	}
}
