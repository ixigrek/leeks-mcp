// leeks-mcp est un serveur MCP (stdio) pour l'API LeekWars : lecture des fiches
// et rapports, lancement de combats.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
	"github.com/ixigrek/leeks-mcp/internal/summary"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "0.2.0"

var errNoToken = errors.New("token LeekWars absent : définir LEEKWARS_TOKEN, LEEKWARS_KEY_FILE ou créer ./key")

func main() {
	log.SetOutput(os.Stderr)
	token, err := leekwars.ResolveToken("key")
	if err != nil {
		log.Fatal(err)
	}
	if token == "" {
		log.Print("aucun token : outils publics seulement")
	}
	server := newServer(leekwars.NewClient(leekwars.DefaultBaseURL, token))
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

// app porte le client et les caches partagés par les outils.
type app struct {
	client *leekwars.Client
	items  *leekwars.ItemsLoader

	mu     sync.Mutex
	fights map[int][]byte // rapports fight/get terminés déjà lus, par id

	pollInterval time.Duration // sondage d'un combat en génération (wait: true)
	pollDeadline time.Duration
}

func newApp(client *leekwars.Client) *app {
	return &app{
		client:       client,
		items:        leekwars.NewItemsLoader(client),
		fights:       map[int][]byte{},
		pollInterval: 2 * time.Second,
		pollDeadline: 60 * time.Second,
	}
}

func newServer(client *leekwars.Client) *mcp.Server {
	return newApp(client).server()
}

var (
	readOnly = &mcp.ToolAnnotations{ReadOnlyHint: true}
	launches = &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: boolPtr(false), OpenWorldHint: boolPtr(true)}
)

func boolPtr(b bool) *bool { return &b }

func (a *app) server() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "leekwars", Version: version}, nil)

	mcp.AddTool(s, &mcp.Tool{Name: "get_leek", Annotations: readOnly, Description: "Fiche d'un poireau : niveau, stats, armes et puces équipées, IA, bilan et 10 derniers combats. Avec token : composants et capital."}, a.getLeek)
	mcp.AddTool(s, &mcp.Tool{Name: "get_farmer", Annotations: readOnly, Description: "Fiche d'un éleveur : poireaux, bilan ; avec token et sans id, l'éleveur du token avec habs, cristaux et inventaire non équipé."}, a.getFarmer)
	mcp.AddTool(s, &mcp.Tool{Name: "list_fights", Annotations: readOnly, Description: "Derniers combats d'un poireau (id, date, résultat, adversaires), filtrables par résultat."}, a.listFights)
	mcp.AddTool(s, &mcp.Tool{Name: "get_fight", Annotations: readOnly, Description: "Rapport d'un combat résumé tour par tour : déplacements, tirs, puces, dégâts, soins, PV. Token requis."}, a.getFight)
	mcp.AddTool(s, &mcp.Tool{Name: "get_fight_logs", Annotations: readOnly, Description: "Lignes debug() des scripts d'un combat, groupées par tour, filtrables par poireau. Token requis."}, a.getFightLogs)
	mcp.AddTool(s, &mcp.Tool{Name: "get_garden", Annotations: readOnly, Description: "État du potager : combats restants (solo/éleveur, équipe), compositions ; avec leek_id, composition_id ou farmer, les adversaires proposés par le matchmaking. Token requis."}, a.getGarden)
	mcp.AddTool(s, &mcp.Tool{Name: "start_solo_fight", Annotations: launches, Description: "Lance un combat solo d'un poireau contre un adversaire proposé par le matchmaking (tiré au sort si target_id absent). Consomme un combat. Renvoie l'id et le status, ou le résumé complet avec wait. Token requis."}, a.startSoloFight)
	mcp.AddTool(s, &mcp.Tool{Name: "start_farmer_fight", Annotations: launches, Description: "Lance un combat d'éleveur (tous les poireaux) contre un éleveur proposé par le matchmaking (tiré au sort si target_id absent). Consomme un combat. Token requis."}, a.startFarmerFight)
	mcp.AddTool(s, &mcp.Tool{Name: "start_team_fight", Annotations: launches, Description: "Lance un combat d'équipe d'une composition contre une composition proposée par le matchmaking (tirée au sort si target_id absent). Consomme un combat d'équipe. Token requis."}, a.startTeamFight)
	mcp.AddTool(s, &mcp.Tool{Name: "start_boss_fight", Annotations: launches, Description: "Lance un combat contre un boss (id ou nom) avec les poireaux donnés, par défaut tous ceux de l'éleveur. Consomme un combat. Token requis."}, a.startBossFight)
	mcp.AddTool(s, &mcp.Tool{Name: "get_item", Annotations: readOnly, Description: "Caractéristiques d'une arme ou d'une puce, par nom (clé anglaise de l'API, ex. laser) ou par id."}, a.getItem)
	return s
}

type rawFlag struct {
	Raw bool `json:"raw,omitempty" jsonschema:"renvoyer la réponse brute de l'API au lieu du résumé"`
}

type leekArgs struct {
	ID int `json:"id" jsonschema:"id du poireau"`
	rawFlag
}

type farmerArgs struct {
	ID int `json:"id,omitempty" jsonschema:"id de l'éleveur ; absent = éleveur du token"`
	rawFlag
}

type listFightsArgs struct {
	LeekID int    `json:"leek_id" jsonschema:"id du poireau"`
	Result string `json:"result,omitempty" jsonschema:"filtre : win, defeat ou draw"`
	Limit  int    `json:"limit,omitempty" jsonschema:"nombre maximal de combats (défaut 10)"`
	rawFlag
}

type fightArgs struct {
	ID     int `json:"id" jsonschema:"id du combat"`
	LeekID int `json:"leek_id,omitempty" jsonschema:"ne garder dans les tours que l'activité de ce poireau"`
	rawFlag
}

type fightLogsArgs struct {
	ID     int `json:"id" jsonschema:"id du combat"`
	LeekID int `json:"leek_id,omitempty" jsonschema:"ne garder que les lignes de ce poireau"`
	rawFlag
}

type itemArgs struct {
	Query string `json:"query" jsonschema:"nom (ex. laser, sun spear) ou id numérique d'une arme ou d'une puce"`
	rawFlag
}

func (a *app) getLeek(ctx context.Context, _ *mcp.CallToolRequest, in leekArgs) (*mcp.CallToolResult, any, error) {
	path := "leek/get/" + strconv.Itoa(in.ID)
	public, err := a.client.Get(ctx, path)
	if err != nil {
		return fail(err)
	}
	var private []byte
	if a.client.HasToken() {
		private, err = a.client.Get(ctx, "leek/get-private/"+strconv.Itoa(in.ID))
		if err != nil {
			log.Printf("leek/get-private ignoré : %v", err)
			private = nil
		}
	}
	if in.Raw {
		return raw(public)
	}
	items, err := a.items.Items(ctx)
	if err != nil {
		return fail(err)
	}
	s, err := summary.Leek(public, private, items)
	if err != nil {
		return fail(err)
	}
	return ok(s)
}

func (a *app) getFarmer(ctx context.Context, _ *mcp.CallToolRequest, in farmerArgs) (*mcp.CallToolResult, any, error) {
	path := "farmer/get-from-token"
	if in.ID != 0 {
		path = "farmer/get/" + strconv.Itoa(in.ID)
	} else if !a.client.HasToken() {
		return fail(errNoToken)
	}
	body, err := a.client.Get(ctx, path)
	if err != nil {
		return fail(err)
	}
	if in.Raw {
		return raw(body)
	}
	items, err := a.items.Items(ctx)
	if err != nil {
		return fail(err)
	}
	s, err := summary.Farmer(body, items)
	if err != nil {
		return fail(err)
	}
	return ok(s)
}

func (a *app) listFights(ctx context.Context, _ *mcp.CallToolRequest, in listFightsArgs) (*mcp.CallToolResult, any, error) {
	body, err := a.client.Get(ctx, "leek/get/"+strconv.Itoa(in.LeekID))
	if err != nil {
		return fail(err)
	}
	if in.Raw {
		var probe struct {
			Fights json.RawMessage `json:"fights"`
		}
		if err := json.Unmarshal(body, &probe); err != nil {
			return fail(err)
		}
		return raw(probe.Fights)
	}
	s, err := summary.FightList(body, in.LeekID, in.Result, in.Limit)
	if err != nil {
		return fail(err)
	}
	return ok(s)
}

func (a *app) fightReport(ctx context.Context, id int) ([]byte, error) {
	if !a.client.HasToken() {
		return nil, errNoToken
	}
	a.mu.Lock()
	body, cached := a.fights[id]
	a.mu.Unlock()
	if cached {
		return body, nil
	}
	body, err := a.client.Get(ctx, "fight/get/"+strconv.Itoa(id))
	if err != nil {
		return nil, err
	}
	a.cacheFight(id, body)
	return body, nil
}

// cacheFight mémorise un rapport, seulement s'il est terminé : un combat encore en
// génération serait sinon figé dans le cache pour toute la session.
func (a *app) cacheFight(id int, body []byte) {
	if st, err := summary.FightStatus(body); err != nil || st != summary.FightFinished {
		return
	}
	a.mu.Lock()
	a.fights[id] = body
	a.mu.Unlock()
}

// finishedReport lit un rapport et refuse ceux encore en génération : résumés,
// leur contenu vide passerait pour un combat sans action.
func (a *app) finishedReport(ctx context.Context, id int) ([]byte, error) {
	body, err := a.fightReport(ctx, id)
	if err != nil {
		return nil, err
	}
	status, err := summary.FightStatus(body)
	if err != nil {
		return nil, err
	}
	if status != summary.FightFinished {
		return nil, fmt.Errorf("combat %d en génération (status %d) : réessayer plus tard", id, status)
	}
	return body, nil
}

func (a *app) getFight(ctx context.Context, _ *mcp.CallToolRequest, in fightArgs) (*mcp.CallToolResult, any, error) {
	if in.Raw {
		body, err := a.fightReport(ctx, in.ID)
		if err != nil {
			return fail(err)
		}
		return raw(body)
	}
	body, err := a.finishedReport(ctx, in.ID)
	if err != nil {
		return fail(err)
	}
	items, err := a.items.Items(ctx)
	if err != nil {
		return fail(err)
	}
	s, err := summary.Fight(body, items, in.LeekID)
	if err != nil {
		return fail(err)
	}
	return ok(s)
}

func (a *app) getFightLogs(ctx context.Context, _ *mcp.CallToolRequest, in fightLogsArgs) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	if in.Raw {
		logs, err := a.client.Get(ctx, "fight/get-logs/"+strconv.Itoa(in.ID))
		if err != nil {
			return fail(err)
		}
		return raw(logs)
	}
	// Le rapport d'abord : il est en cache et refuse un combat encore en génération.
	report, err := a.finishedReport(ctx, in.ID)
	if err != nil {
		return fail(err)
	}
	logs, err := a.client.Get(ctx, "fight/get-logs/"+strconv.Itoa(in.ID))
	if err != nil {
		return fail(err)
	}
	s, err := summary.Logs(logs, report, in.LeekID)
	if err != nil {
		return fail(err)
	}
	return ok(s)
}

func (a *app) getItem(ctx context.Context, _ *mcp.CallToolRequest, in itemArgs) (*mcp.CallToolResult, any, error) {
	items, err := a.items.Items(ctx)
	if err != nil {
		return fail(err)
	}
	matches := items.Find(in.Query)
	if len(matches) == 0 {
		return fail(fmt.Errorf("aucune arme ni puce ne correspond à %q", in.Query))
	}
	if in.Raw {
		var out []any
		for _, m := range matches {
			if m.Weapon != nil {
				out = append(out, m.Weapon)
			} else {
				out = append(out, m.Chip)
			}
		}
		return ok(out)
	}
	if len(matches) == 1 {
		return ok(summary.Item(matches[0]))
	}
	out := make([]summary.ItemSummary, 0, len(matches))
	for _, m := range matches {
		out = append(out, summary.Item(m))
	}
	return ok(map[string]any{"ambiguous": true, "candidates": out})
}

// ok sérialise le résumé en JSON compact : moins de tokens qu'une sortie indentée.
func ok(v any) (*mcp.CallToolResult, any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return fail(err)
	}
	return text(string(data)), nil, nil
}

// raw renvoie la réponse de l'API telle quelle, recompactée.
func raw(body []byte) (*mcp.CallToolResult, any, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, body); err != nil {
		return text(strings.TrimSpace(string(body))), nil, nil
	}
	return text(buf.String()), nil, nil
}

func fail(err error) (*mcp.CallToolResult, any, error) {
	r := text(err.Error())
	r.IsError = true
	return r, nil, nil
}

func text(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}
