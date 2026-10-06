package main

import (
	"encoding/json"
	"strings"
	"testing"
)

var loadoutRoutes = map[string][]string{
	"GET /api/loadout/get-all":   {"loadouts.json"},
	"POST /api/loadout/create":   {"loadout_create.json"},
	"PUT /api/loadout/update":    {"loadout_update.json"},
	"POST /api/loadout/apply":    {"loadout_apply.json"},
	"DELETE /api/loadout/delete": {"loadout_delete.json"},
}

func TestListLoadouts(t *testing.T) {
	cs, _ := sessionAPI(t, "tok", loadoutRoutes)
	text, isErr := call(t, cs, "list_loadouts", map[string]any{})
	if isErr || !strings.Contains(text, `"name":"mcp-test"`) || !strings.Contains(text, `"capital":1370`) || !strings.Contains(text, `"owned_weapons":["pistol"`) {
		t.Fatalf("loadouts : %.400s", text)
	}
	raw, isErr := call(t, cs, "list_loadouts", map[string]any{"raw": true})
	if isErr || !strings.Contains(raw, `"owned_component_instances"`) {
		t.Fatalf("brut : %.300s", raw)
	}
}

func TestSaveLoadoutFromLeek(t *testing.T) {
	cs, api := sessionAPI(t, "tok", loadoutRoutes)
	text, isErr := call(t, cs, "save_loadout", map[string]any{
		"name": "build", "from_leek_id": 135146, "weapons": []string{"laser", "pistol"}, "stats": map[string]any{"strength": 703, "ram": 0, "agility": 20},
	})
	if isErr || !strings.Contains(text, `"id":1485`) || !strings.Contains(text, `"name":"mcp-test"`) {
		t.Fatalf("création : %.400s", text)
	}
	post := api.lastPost(t)
	if post.Method != "POST" || post.Path != "/api/loadout/create" || post.Body["name"] != "build" || post.Body["icon"] != "strength" {
		t.Fatalf("POST = %+v", post)
	}
	if post.Body["weapons"] != "[42,37]" || post.Body["chips"] != "[14,6,20,96,22,32,31,110,35,25,16,33]" {
		t.Fatalf("équipement envoyé : %v / %v", post.Body["weapons"], post.Body["chips"])
	}
	var stats map[string]int
	json.Unmarshal([]byte(post.Body["stats"].(string)), &stats)
	// Préremplissage : life 210, strength 700, tp 255, mp 60, ram 50 ; strength et agility remplacées, ram retirée.
	want := map[string]int{"life": 210, "strength": 703, "tp": 255, "mp": 60, "agility": 20}
	if len(stats) != len(want) {
		t.Fatalf("stats envoyées : %v", stats)
	}
	for k, v := range want {
		if stats[k] != v {
			t.Fatalf("stats envoyées : %v, attendu %v", stats, want)
		}
	}
	var comps []map[string]any
	json.Unmarshal([]byte(post.Body["components"].(string)), &comps)
	if len(comps) != 6 || comps[5]["index"] != float64(5) || comps[5]["template"] != float64(313) || comps[0]["stats"] != nil {
		t.Fatalf("composants envoyés : %v", post.Body["components"])
	}
}

func TestSaveLoadoutUpdatesExisting(t *testing.T) {
	cs, api := sessionAPI(t, "tok", loadoutRoutes)
	text, isErr := call(t, cs, "save_loadout", map[string]any{"set_id": 1485, "icon": "agility", "chips": []string{"flash", "shock"}})
	if isErr || !strings.Contains(text, `"name":"mcp-test-2"`) || !strings.Contains(text, `"icon":"agility"`) {
		t.Fatalf("mise à jour : %.400s", text)
	}
	put := api.lastPost(t)
	if put.Method != "PUT" || put.Path != "/api/loadout/update" || put.Body["set_id"] != float64(1485) || put.Body["name"] != "mcp-test" {
		t.Fatalf("PUT = %+v", put)
	}
	if put.Body["chips"] != "[6,1]" || put.Body["weapons"] != "[42,43,153,180]" || !strings.Contains(put.Body["stats"].(string), `"strength":700`) {
		t.Fatalf("contenu conservé : %+v", put.Body)
	}
	text, isErr = call(t, cs, "save_loadout", map[string]any{"set_id": 1, "name": "x"})
	if !isErr || !strings.Contains(text, "loadout 1 introuvable") {
		t.Fatalf("loadout inconnu : %s", text)
	}
}

func TestSaveLoadoutValidatesBeforeSending(t *testing.T) {
	cs, api := sessionAPI(t, "tok", loadoutRoutes)
	cases := []struct {
		args map[string]any
		want string
	}{
		{map[string]any{"weapons": []string{"laser"}}, "name requis"},
		{map[string]any{"name": "x", "weapons": []string{"plop"}}, `arme "plop" inconnue`},
		{map[string]any{"name": "x", "weapons": []string{"laser", "laser"}}, "laser en double"},
		{map[string]any{"name": "x", "chips": []string{"laser"}}, `puce "laser" inconnue`},
		{map[string]any{"name": "x", "stats": map[string]any{"speed": 1}}, `"speed" inconnue`},
		{map[string]any{"name": "x", "stats": map[string]any{"life": -1}}, "capital négatif"},
		{map[string]any{"name": "x", "from_leek_id": 135146, "stats": map[string]any{"agility": 1000}}, "2275 capital demandé, 1300 au total"},
	}
	for _, c := range cases {
		text, isErr := call(t, cs, "save_loadout", c.args)
		if !isErr || !strings.Contains(text, c.want) {
			t.Fatalf("%v : %q attendu, eu %s", c.args, c.want, text)
		}
	}
	if n := len(api.writes()); n != 0 {
		t.Fatalf("%d écritures envoyées malgré les refus", n)
	}
}

func TestApplyLoadout(t *testing.T) {
	cs, api := sessionAPI(t, "tok", loadoutRoutes)
	text, isErr := call(t, cs, "apply_loadout", map[string]any{"set_id": 1485, "leek_id": 135146})
	if isErr || !strings.Contains(text, `"weapons":["laser","rhino"`) || !strings.Contains(text, `"components":6`) || !strings.Contains(text, `"stats_changed":false`) {
		t.Fatalf("application : %.400s", text)
	}
	post := api.lastPost(t)
	if post.Path != "/api/loadout/apply" || post.Body["set_id"] != float64(1485) || post.Body["leek_id"] != float64(135146) || post.Body["use_restat"] != false {
		t.Fatalf("POST = %+v", post)
	}
}

func TestApplyLoadoutRelaysCapitalError(t *testing.T) {
	routes := map[string][]string{}
	for k, v := range loadoutRoutes {
		routes[k] = v
	}
	routes["POST /api/loadout/apply"] = []string{"400:loadout_apply_error.json"}
	cs, _ := sessionAPI(t, "tok", routes)
	text, isErr := call(t, cs, "apply_loadout", map[string]any{"set_id": 1486, "leek_id": 135146, "use_restat": true})
	if !isErr || !strings.Contains(text, "not_enough_capital") || !strings.Contains(text, `"available":1375`) {
		t.Fatalf("erreur de capital attendue : %s", text)
	}
}

func TestDeleteLoadout(t *testing.T) {
	cs, api := sessionAPI(t, "tok", loadoutRoutes)
	text, isErr := call(t, cs, "delete_loadout", map[string]any{"set_id": 1485})
	if isErr || !strings.Contains(text, `"deleted":1485`) {
		t.Fatalf("suppression : %s", text)
	}
	del := api.lastPost(t)
	if del.Method != "DELETE" || del.Path != "/api/loadout/delete" || del.Body["set_id"] != float64(1485) {
		t.Fatalf("DELETE = %+v", del)
	}
}

func TestLoadoutToolsRequireToken(t *testing.T) {
	cs, _ := sessionAPI(t, "", loadoutRoutes)
	calls := map[string]map[string]any{
		"list_loadouts":  {},
		"save_loadout":   {"name": "x"},
		"apply_loadout":  {"set_id": 1485, "leek_id": 135146},
		"delete_loadout": {"set_id": 1485},
	}
	for tool, args := range calls {
		text, isErr := call(t, cs, tool, args)
		if !isErr || !strings.Contains(text, "LEEKWARS_TOKEN") {
			t.Fatalf("%s : erreur de token attendue : %s", tool, text)
		}
	}
}
