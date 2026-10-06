package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fakeAPI sert les fixtures de testdata/ comme l'API LeekWars. Les routes sont
// typées méthode + chemin ; une route peut servir une séquence de fixtures (la
// i-ème requête reçoit la i-ème, puis la dernière se répète), chacune précédée
// d'un code HTTP optionnel ("404:fichier.json"). Les corps POST sont
// (PUT, DELETE) enregistrés pour vérifier ce que les outils envoient.
type fakeAPI struct {
	*httptest.Server
	mu    sync.Mutex
	hits  map[string]int
	posts []postRecord
}

type postRecord struct {
	Method string
	Path   string
	Body   map[string]any
}

var fixtureRoutes = map[string][]string{
	"GET /api/leek/get/135146":                 {"leek_135146.json"},
	"GET /api/leek/get-private/135146":         {"leek_private_135146.json"},
	"GET /api/farmer/get/128381":               {"farmer_128381.json"},
	"GET /api/farmer/get-from-token":           {"farmer_token.json"},
	"GET /api/weapon/get-all":                  {"weapons.json"},
	"GET /api/chip/get-all":                    {"chips.json"},
	"GET /api/fight/get/53988601":              {"fight_53988601.json"},
	"GET /api/fight/get-logs/53988601":         {"logs_53988601.json"},
	"GET /api/boss/get-all":                    {"bosses.json"},
	"GET /api/history/get-leek-history/135146": {"leek_history_135146.json"},
}

func newFakeAPI(t *testing.T, extra map[string][]string) *fakeAPI {
	t.Helper()
	routes := map[string][]string{}
	for k, v := range fixtureRoutes {
		routes[k] = v
	}
	for k, v := range extra {
		routes[k] = v
	}
	f := &fakeAPI{hits: map[string]int{}}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		f.mu.Lock()
		n := f.hits[key]
		f.hits[key] = n + 1
		if r.Method != http.MethodGet {
			rec := postRecord{Method: r.Method, Path: r.URL.Path}
			json.NewDecoder(r.Body).Decode(&rec.Body)
			f.posts = append(f.posts, rec)
		}
		f.mu.Unlock()
		seq, ok := routes[key]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"success":false,"error":"not_found"}`))
			return
		}
		if n >= len(seq) {
			n = len(seq) - 1
		}
		name := seq[n]
		// "404:fichier.json" sert la fixture avec ce code HTTP (erreurs de lancement).
		if code, rest, found := strings.Cut(name, ":"); found {
			status, err := strconv.Atoi(code)
			if err != nil {
				t.Fatalf("route %s : code HTTP %q invalide", key, code)
			}
			w.WriteHeader(status)
			name = rest
		}
		data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		w.Write(data)
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeAPI) count(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.hits[key]
}

// writes renvoie, dans l'ordre, les requêtes d'écriture reçues.
func (f *fakeAPI) writes() []postRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]postRecord(nil), f.posts...)
}

func (f *fakeAPI) lastPost(t *testing.T) postRecord {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.posts) == 0 {
		t.Fatal("aucun POST reçu")
	}
	return f.posts[len(f.posts)-1]
}

// sessionAPI ouvre une session MCP en mémoire sur une fausse API, avec des délais
// de sondage raccourcis.
func sessionAPI(t *testing.T, token string, extra map[string][]string) (*mcp.ClientSession, *fakeAPI) {
	t.Helper()
	api := newFakeAPI(t, extra)
	a := newApp(leekwars.NewClient(api.URL, token))
	a.pollInterval = 5 * time.Millisecond
	a.pollDeadline = time.Second
	a.launchInterval = time.Millisecond
	server := a.server()
	st, ct := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := server.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs, api
}

func session(t *testing.T, token string) *mcp.ClientSession {
	t.Helper()
	cs, _ := sessionAPI(t, token, nil)
	return cs
}

func call(t *testing.T, cs *mcp.ClientSession, tool string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String(), res.IsError
}

func TestFightCacheSkipsPendingReports(t *testing.T) {
	cs, api := sessionAPI(t, "tok", map[string][]string{
		"GET /api/fight/get/53988601": {"fight_pending.json", "fight_pending.json", "fight_53988601.json"},
	})
	for i := 0; i < 2; i++ {
		text, isErr := call(t, cs, "get_fight", map[string]any{"id": 53988601, "raw": true})
		if isErr || !strings.Contains(text, `"status":0`) {
			t.Fatalf("rapport en attente attendu : %.200s", text)
		}
	}
	text, isErr := call(t, cs, "get_fight", map[string]any{"id": 53988601, "raw": true})
	if isErr || !strings.Contains(text, `"status":2`) {
		t.Fatalf("rapport terminé attendu : %.200s", text)
	}
	call(t, cs, "get_fight", map[string]any{"id": 53988601})
	if n := api.count("GET /api/fight/get/53988601"); n != 3 {
		t.Fatalf("%d requêtes fight/get, attendu 3 (2 en attente non cachées, 1 terminée cachée)", n)
	}
}

func TestToolsAreAnnotated(t *testing.T) {
	cs := session(t, "tok")
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range res.Tools {
		if tool.Annotations == nil {
			t.Fatalf("outil %s sans annotations", tool.Name)
		}
		write := strings.HasPrefix(tool.Name, "start_") || strings.HasSuffix(tool.Name, "_loadout") || tool.Name == "ai_push" || tool.Name == "run_batch"
		if tool.Annotations.ReadOnlyHint == write {
			t.Fatalf("outil %s : ReadOnlyHint = %v", tool.Name, tool.Annotations.ReadOnlyHint)
		}
	}
}

func TestListsTwentyTools(t *testing.T) {
	cs := session(t, "tok")
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}
	want := []string{"get_leek", "get_farmer", "list_fights", "get_fight", "get_fight_logs", "fight_stats", "get_item",
		"get_garden", "start_solo_fight", "start_farmer_fight", "start_team_fight", "start_boss_fight", "run_batch",
		"list_loadouts", "save_loadout", "apply_loadout", "delete_loadout", "ai_tree", "ai_read", "ai_push"}
	if len(res.Tools) != len(want) {
		t.Fatalf("%d outils, attendu %d : %v", len(res.Tools), len(want), names)
	}
	for _, want := range want {
		if !names[want] {
			t.Fatalf("outil %s absent : %v", want, names)
		}
	}
}

func TestGetLeekSummaryAndRaw(t *testing.T) {
	cs := session(t, "tok")
	text, isErr := call(t, cs, "get_leek", map[string]any{"id": 135146})
	if isErr {
		t.Fatal(text)
	}
	var s struct {
		Name    string `json:"name"`
		Capital *int   `json:"capital"`
	}
	if err := json.Unmarshal([]byte(text), &s); err != nil || s.Name != "plop2point0" || s.Capital == nil {
		t.Fatalf("résumé : %s", text)
	}
	raw, isErr := call(t, cs, "get_leek", map[string]any{"id": 135146, "raw": true})
	if isErr || !strings.Contains(raw, `"talent_history"`) {
		t.Fatalf("brut attendu : %.200s", raw)
	}
}

func TestListFightsUsesFullHistory(t *testing.T) {
	cs := session(t, "")
	text, isErr := call(t, cs, "list_fights", map[string]any{"leek_id": 135146, "result": "defeat", "limit": 2})
	if isErr || !strings.Contains(text, `"id":53996448`) || !strings.Contains(text, `"id":53996167`) {
		t.Fatalf("défaites : %.300s", text)
	}
	raw, isErr := call(t, cs, "list_fights", map[string]any{"leek_id": 135146, "result": "defeat", "limit": 2, "raw": true})
	if isErr || strings.Count(raw, `"result":"defeat"`) != 2 || strings.Contains(raw, `"result":"win"`) {
		t.Fatalf("brut : %.300s", raw)
	}
}

func TestReadToolsTurnRangeQueriesAndItems(t *testing.T) {
	cs := session(t, "tok")
	text, isErr := call(t, cs, "get_fight", map[string]any{"id": 53988601, "from_turn": 2, "to_turn": 3})
	var fight struct {
		Duration int `json:"duration"`
		Turns    []struct {
			Turn int `json:"turn"`
		} `json:"turns"`
	}
	if err := json.Unmarshal([]byte(text), &fight); isErr || err != nil || fight.Duration != 41 ||
		len(fight.Turns) != 2 || fight.Turns[0].Turn != 2 || fight.Turns[1].Turn != 3 {
		t.Fatalf("get_fight tours 2-3 : %.300s", text)
	}
	text, isErr = call(t, cs, "get_fight_logs", map[string]any{"id": 53988601, "from_turn": 2, "to_turn": 2})
	if isErr || !strings.Contains(text, `"turn":2`) || strings.Contains(text, `"turn":3`) || strings.Contains(text, `"turn":1,`) {
		t.Fatalf("get_fight_logs tour 2 : %.300s", text)
	}
	text, isErr = call(t, cs, "get_fight_logs", map[string]any{"id": 53988601, "from_turn": 5, "to_turn": 2})
	if !isErr || !strings.Contains(text, "invalides") {
		t.Fatalf("tours inversés : %s", text)
	}
	text, isErr = call(t, cs, "get_item", map[string]any{"queries": []string{"laser", "25", "objet_imaginaire"}})
	if isErr || !strings.Contains(text, `{"query":"laser","result":{"kind":"weapon"`) ||
		!strings.Contains(text, `{"query":"25","result":{"ambiguous":true`) ||
		!strings.Contains(text, `{"query":"objet_imaginaire","error":"aucune arme ni puce`) {
		t.Fatalf("queries : %.600s", text)
	}
	text, isErr = call(t, cs, "get_item", map[string]any{})
	if !isErr || !strings.Contains(text, "query ou queries requis") {
		t.Fatalf("sans requête : %s", text)
	}
	text, isErr = call(t, cs, "get_leek", map[string]any{"id": 135146, "items": true})
	if isErr || !strings.Contains(text, `"name":"leather_boots","details":{"kind":"chip"`) || !strings.Contains(text, `"effects":[`) {
		t.Fatalf("get_leek items : %.600s", text)
	}
}

func TestFightStats(t *testing.T) {
	cs := session(t, "tok")
	text, isErr := call(t, cs, "fight_stats", map[string]any{"id": 53988601, "leek_id": 135146})
	if isErr || !strings.Contains(text, `"name":"plop2point0"`) || !strings.Contains(text, `"turns":[{"turn":1`) || !strings.Contains(text, `"flags":`) {
		t.Fatalf("diagnostic : %.400s", text)
	}
	text, isErr = call(t, cs, "fight_stats", map[string]any{"id": 53988601})
	if !isErr || !strings.Contains(text, "leek_id") {
		t.Fatalf("leek_id manquant : %s", text)
	}
}

func TestGetFightRequiresToken(t *testing.T) {
	cs := session(t, "")
	text, isErr := call(t, cs, "get_fight", map[string]any{"id": 53988601})
	if !isErr || !strings.Contains(text, "LEEKWARS_TOKEN") {
		t.Fatalf("erreur de token attendue : %s", text)
	}
}

func TestGetFightLogsFiltered(t *testing.T) {
	cs := session(t, "tok")
	text, isErr := call(t, cs, "get_fight_logs", map[string]any{"id": 53988601, "leek_id": 135146})
	if isErr {
		t.Fatal(text)
	}
	if !strings.Contains(text, "profil : burst") {
		t.Fatalf("logs : %.300s", text)
	}
}

func TestGetItemByNameAndUnknown(t *testing.T) {
	cs := session(t, "")
	text, isErr := call(t, cs, "get_item", map[string]any{"query": "Laser"})
	if isErr || !strings.Contains(text, `"item":42`) {
		t.Fatalf("laser : %.300s", text)
	}
	text, isErr = call(t, cs, "get_item", map[string]any{"query": "25"})
	if isErr || !strings.Contains(text, `"ambiguous":true`) || !strings.Contains(text, `"name":"lightninger"`) || !strings.Contains(text, `"name":"steroid"`) || !strings.Contains(text, `"matched_by":"id"`) {
		t.Fatalf("25 ambigu : %.400s", text)
	}
	text, isErr = call(t, cs, "get_item", map[string]any{"query": "25", "kind": "chip"})
	if isErr || strings.Contains(text, "ambiguous") || !strings.Contains(text, `"name":"steroid"`) {
		t.Fatalf("25 puce : %.400s", text)
	}
	text, isErr = call(t, cs, "get_item", map[string]any{"query": "objet_imaginaire"})
	if !isErr {
		t.Fatalf("erreur attendue : %s", text)
	}
}

func TestGetFarmerWithoutIDUsesToken(t *testing.T) {
	cs := session(t, "tok")
	text, isErr := call(t, cs, "get_farmer", map[string]any{})
	if isErr || !strings.Contains(text, `"habs":11635402`) {
		t.Fatalf("farmer : %.300s", text)
	}
}

func TestAPIErrorIsReportedAsToolError(t *testing.T) {
	cs := session(t, "tok")
	text, isErr := call(t, cs, "get_fight", map[string]any{"id": 1})
	if !isErr || !strings.Contains(text, "not_found") {
		t.Fatalf("erreur API attendue : %s", text)
	}
}
