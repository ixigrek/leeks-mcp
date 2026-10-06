package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fakeAPI sert les fixtures de testdata/ comme l'API LeekWars.
func fakeAPI(t *testing.T) *httptest.Server {
	t.Helper()
	routes := map[string]string{
		"/api/leek/get/135146":         "leek_135146.json",
		"/api/leek/get-private/135146": "leek_private_135146.json",
		"/api/farmer/get/128381":       "farmer_128381.json",
		"/api/farmer/get-from-token":   "farmer_token.json",
		"/api/weapon/get-all":          "weapons.json",
		"/api/chip/get-all":            "chips.json",
		"/api/fight/get/53988601":      "fight_53988601.json",
		"/api/fight/get-logs/53988601": "logs_53988601.json",
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"success":false,"error":"not_found"}`))
			return
		}
		data, err := os.ReadFile(filepath.Join("..", "..", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		w.Write(data)
	}))
}

func session(t *testing.T, token string) *mcp.ClientSession {
	t.Helper()
	api := fakeAPI(t)
	t.Cleanup(api.Close)
	server := newServer(leekwars.NewClient(api.URL, token))
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

func TestListsSixTools(t *testing.T) {
	cs := session(t, "tok")
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"get_leek", "get_farmer", "list_fights", "get_fight", "get_fight_logs", "get_item"} {
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
