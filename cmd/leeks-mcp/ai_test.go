package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ixigrek/leeks-mcp/internal/summary"
)

// leekFile écrit un fichier .leek dont le code est celui de la fixture ai/read.
func leekFile(t *testing.T, dir, name, fixture string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	code, _, err := summary.AIRead(data)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func pushResult(t *testing.T, text string) aiPushResult {
	t.Helper()
	var r aiPushResult
	if err := json.Unmarshal([]byte(text), &r); err != nil {
		t.Fatalf("%v : %s", err, text)
	}
	return r
}

// writePaths renvoie les IA écrites par ai/write, dans l'ordre.
func writePaths(api *fakeAPI) []string {
	var out []string
	for _, p := range api.writes() {
		if p.Path == "/api/ai/write" {
			out = append(out, p.Body["path"].(string))
		}
	}
	return out
}

var aiTreeRoute = map[string][]string{"GET /api/ai/get-farmer-tree": {"ai_tree.json"}}

func aiRoutes(extra map[string][]string) map[string][]string {
	routes := map[string][]string{}
	for k, v := range aiTreeRoute {
		routes[k] = v
	}
	for k, v := range extra {
		routes[k] = v
	}
	return routes
}

func TestAIPushOrdersLibrariesFirst(t *testing.T) {
	cs, api := sessionAPI(t, "tok", aiRoutes(map[string][]string{
		// Pour chaque fichier : lecture avant (ancien code), puis relecture (nouveau).
		"POST /api/ai/read":  {"ai_read_v1.json", "ai_read_v2.json", "ai_read_v1.json", "ai_read_v2.json", "ai_read_v1.json", "ai_read_v2.json"},
		"POST /api/ai/write": {"ai_write_force_grp.json", "ai_write_force_nasu.json", "ai_write_force.json"},
	}))
	dir := t.TempDir()
	files := []string{
		leekFile(t, dir, "force.leek", "ai_read_v2.json"),
		leekFile(t, dir, "force_nasu.leek", "ai_read_v2.json"),
		leekFile(t, dir, "force_grp.leek", "ai_read_v2.json"),
	}
	text, isErr := call(t, cs, "ai_push", map[string]any{"files": files})
	if isErr {
		t.Fatal(text)
	}
	if got := strings.Join(writePaths(api), ","); got != "force_grp,force_nasu,force" {
		t.Fatalf("ordre d'écriture : %s", got)
	}
	r := pushResult(t, text)
	for _, f := range r.Files {
		if f.Status != pushPushed || f.OldVersion != "A" || f.NewVersion != "B" || f.VersionUnchanged {
			t.Fatalf("fichier : %+v", f)
		}
	}
	if r.Files[0].Name != "force_grp" || !r.Files[0].Includers["force"] {
		t.Fatalf("includers du _grp : %+v", r.Files[0])
	}
	if p := r.Files[2].PlayedBy; len(p) != 1 || p[0] != "135146" {
		t.Fatalf("played_by de force : %v", p)
	}
	if len(r.Warnings) != 0 {
		t.Fatalf("avertissements : %v", r.Warnings)
	}
	body := api.writes()
	var sent string
	for _, p := range body {
		if p.Path == "/api/ai/write" {
			sent = p.Body["code"].(string)
		}
	}
	if !strings.Contains(sent, `VERSION = "B"`) {
		t.Fatalf("code envoyé : %q", sent)
	}
}

func TestAIPushRejectsBadNamesBeforeWriting(t *testing.T) {
	cs, api := sessionAPI(t, "tok", aiRoutes(nil))
	dir := t.TempDir()
	cases := map[string][]string{
		"extension":  {leekFile(t, dir, "force.leek.leek", "ai_read_v2.json")},
		"pas .leek":  {leekFile(t, dir, "force.txt", "ai_read_v2.json")},
		"doublon":    {leekFile(t, dir, "force.leek", "ai_read_v2.json"), filepath.Join(dir, ".", "force.leek")},
		"inexistant": {filepath.Join(dir, "absent.leek")},
		"hors ligne": {leekFile(t, dir, "nouvelle.leek", "ai_read_v2.json")},
	}
	for label, files := range cases {
		text, isErr := call(t, cs, "ai_push", map[string]any{"files": files})
		if !isErr {
			t.Fatalf("%s : erreur attendue : %s", label, text)
		}
	}
	if n := len(api.writes()); n != 0 {
		t.Fatalf("%d requêtes POST, attendu 0", n)
	}
}

func TestAIPushSkipsIdenticalAndFlagsVersion(t *testing.T) {
	cs, api := sessionAPI(t, "tok", aiRoutes(map[string][]string{
		// force_grp déjà identique ; force modifié sans changer VERSION.
		"POST /api/ai/read":  {"ai_read_v2.json", "ai_read_v1_same_version.json", "ai_read_v2.json"},
		"POST /api/ai/write": {"ai_write_force.json"},
	}))
	dir := t.TempDir()
	files := []string{leekFile(t, dir, "force.leek", "ai_read_v2.json"), leekFile(t, dir, "force_grp.leek", "ai_read_v2.json")}
	text, isErr := call(t, cs, "ai_push", map[string]any{"files": files})
	if isErr {
		t.Fatal(text)
	}
	r := pushResult(t, text)
	if r.Files[0].Status != pushUnchanged || r.Files[1].Status != pushPushed || !r.Files[1].VersionUnchanged {
		t.Fatalf("statuts : %s", text)
	}
	if len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "force") {
		t.Fatalf("avertissements : %v", r.Warnings)
	}
	if got := strings.Join(writePaths(api), ","); got != "force" {
		t.Fatalf("écritures : %s", got)
	}
}

func TestAIPushStopsOnProblems(t *testing.T) {
	cs, api := sessionAPI(t, "tok", aiRoutes(map[string][]string{
		"POST /api/ai/read":  {"ai_read_v1.json"},
		"POST /api/ai/write": {"ai_write_paladin_nasu_problems.json"},
	}))
	dir := t.TempDir()
	files := []string{leekFile(t, dir, "paladin.leek", "ai_read_v2.json"), leekFile(t, dir, "paladin_nasu.leek", "ai_read_v2.json")}
	text, isErr := call(t, cs, "ai_push", map[string]any{"files": files})
	if !isErr {
		t.Fatalf("erreur attendue : %s", text)
	}
	r := pushResult(t, text)
	if r.Files[0].Name != "paladin_nasu" || r.Files[0].Status != pushProblems || len(r.Files[0].Problems) != 3 {
		t.Fatalf("paladin_nasu : %+v", r.Files[0])
	}
	if r.Files[1].Status != pushSkipped {
		t.Fatalf("paladin : %+v", r.Files[1])
	}
	if got := strings.Join(writePaths(api), ","); got != "paladin_nasu" {
		t.Fatalf("écritures : %s", got)
	}
}

func TestAIPushDetectsMismatch(t *testing.T) {
	cs, _ := sessionAPI(t, "tok", aiRoutes(map[string][]string{
		// La relecture renvoie encore l'ancien code.
		"POST /api/ai/read":  {"ai_read_v1.json"},
		"POST /api/ai/write": {"ai_write_force.json"},
	}))
	file := leekFile(t, t.TempDir(), "force.leek", "ai_read_v2.json")
	text, isErr := call(t, cs, "ai_push", map[string]any{"files": []string{file}})
	if !isErr {
		t.Fatalf("erreur attendue : %s", text)
	}
	if f := pushResult(t, text).Files[0]; f.Status != pushMismatch || f.FirstDiffLine != 1 {
		t.Fatalf("force : %+v", f)
	}
}

func TestAIRead(t *testing.T) {
	cs, _ := sessionAPI(t, "tok", aiRoutes(map[string][]string{
		"POST /api/ai/read": {"ai_read_force_nasu.json"},
	}))
	text, isErr := call(t, cs, "ai_read", map[string]any{"name": "force_nasu"})
	var r aiReadResult
	if isErr || json.Unmarshal([]byte(text), &r) != nil || !strings.HasPrefix(r.Version, "N-") || r.Code == "" {
		t.Fatalf("lecture : %.300s", text)
	}
	file := leekFile(t, t.TempDir(), "force_nasu.leek", "ai_read_force_nasu.json")
	text, isErr = call(t, cs, "ai_read", map[string]any{"name": "force_nasu", "file": file})
	var c aiCompareResult
	if isErr || json.Unmarshal([]byte(text), &c) != nil || !c.Identical || c.LocalVersion != r.Version || strings.Contains(text, "code") {
		t.Fatalf("comparaison : %s", text)
	}
	if text, isErr := call(t, cs, "ai_read", map[string]any{"name": "force_nasu.leek"}); !isErr {
		t.Fatalf("nom .leek refusé attendu : %s", text)
	}
}

func TestAIReadUnknown(t *testing.T) {
	cs, _ := sessionAPI(t, "tok", aiRoutes(map[string][]string{
		"POST /api/ai/read": {"400:ai_read_error.json"},
	}))
	text, isErr := call(t, cs, "ai_read", map[string]any{"name": "inexistant"})
	if !isErr || !strings.Contains(text, "file_not_found") {
		t.Fatalf("erreur attendue : %s", text)
	}
}

func TestAITree(t *testing.T) {
	cs, _ := sessionAPI(t, "tok", aiTreeRoute)
	text, isErr := call(t, cs, "ai_tree", nil)
	if isErr || !strings.Contains(text, `"135146":"force"`) || strings.Contains(text, `"bin"`) {
		t.Fatalf("arbre : %.300s", text)
	}
}
