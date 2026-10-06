package main

import (
	"strconv"
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

func TestStartSoloFightPicksRandomOpponent(t *testing.T) {
	cs, api := sessionAPI(t, "tok", gardenRoutes)
	text, isErr := call(t, cs, "start_solo_fight", map[string]any{"leek_id": 135146})
	if isErr || !strings.Contains(text, `"fight_id":53994496`) || !strings.Contains(text, `"status":0`) {
		t.Fatalf("lancement : %.300s", text)
	}
	post := api.lastPost(t)
	if post.Path != "/api/garden/start-solo-fight" || post.Body["leek_id"] != float64(135146) {
		t.Fatalf("POST = %+v", post)
	}
	target := post.Body["target_id"].(float64)
	proposed := map[float64]bool{83810: true, 130864: true, 58421: true, 107570: true, 130787: true}
	if !proposed[target] {
		t.Fatalf("target_id %v hors des adversaires proposés", target)
	}
	if !strings.Contains(text, `"target_id":`+strconv.Itoa(int(target))) || !strings.Contains(text, `"target_name":`) {
		t.Fatalf("cible tirée absente de la réponse : %.300s", text)
	}
}

func TestStartSoloFightWithTargetAndWait(t *testing.T) {
	cs, api := sessionAPI(t, "tok", gardenRoutes)
	text, isErr := call(t, cs, "start_solo_fight", map[string]any{"leek_id": 135146, "target_id": 83810, "wait": true})
	if isErr || !strings.Contains(text, `"id":53994496`) || !strings.Contains(text, `"turns"`) {
		t.Fatalf("résumé attendu : %.300s", text)
	}
	if strings.Contains(text, `"target_name"`) {
		t.Fatalf("cible explicite : pas de target_name attendu : %.300s", text)
	}
	if post := api.lastPost(t); post.Body["target_id"] != float64(83810) {
		t.Fatalf("POST = %+v", post)
	}
	if n := api.count("GET /api/fight/get/53994496"); n < 2 {
		t.Fatalf("%d sondage(s), au moins 2 attendus (en attente puis terminé)", n)
	}
	if n := api.count("GET /api/garden/get-leek-opponents/135146"); n != 0 {
		t.Fatalf("adversaires lus %d fois alors que la cible est fournie", n)
	}
	// Le rapport terminé est désormais en cache : get_fight ne refait pas de requête.
	before := api.count("GET /api/fight/get/53994496")
	call(t, cs, "get_fight", map[string]any{"id": 53994496})
	if api.count("GET /api/fight/get/53994496") != before {
		t.Fatal("rapport terminé non mis en cache après wait")
	}
}

func TestStartFightRequiresToken(t *testing.T) {
	cs, _ := sessionAPI(t, "", gardenRoutes)
	for tool, args := range map[string]map[string]any{
		"start_solo_fight":   {"leek_id": 135146},
		"start_farmer_fight": {},
		"start_team_fight":   {"composition_id": 7},
		"start_boss_fight":   {"boss": "nasu_samurai"},
	} {
		text, isErr := call(t, cs, tool, args)
		if !isErr || !strings.Contains(text, "LEEKWARS_TOKEN") {
			t.Fatalf("%s : erreur de token attendue : %s", tool, text)
		}
	}
}
