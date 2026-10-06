package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/ixigrek/leeks-mcp/internal/summary"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type gardenArgs struct {
	LeekID        int  `json:"leek_id,omitempty" jsonschema:"ajouter les adversaires solo proposés à ce poireau"`
	CompositionID int  `json:"composition_id,omitempty" jsonschema:"ajouter les compositions adverses proposées à cette composition"`
	Farmer        bool `json:"farmer,omitempty" jsonschema:"ajouter les éleveurs adverses proposés"`
	rawFlag
}

// gardenResult est la réponse de get_garden : l'état du potager et, à la demande,
// les adversaires proposés pour un poireau, une composition ou l'éleveur.
type gardenResult struct {
	Garden       *summary.GardenSummary `json:"garden"`
	OpponentsFor string                 `json:"opponents_for,omitempty"`
	Opponents    []summary.Opponent     `json:"opponents,omitempty"`
}

func (a *app) getGarden(ctx context.Context, _ *mcp.CallToolRequest, in gardenArgs) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	var path, label string
	selectors := 0
	if in.LeekID != 0 {
		selectors++
		path, label = "garden/get-leek-opponents/"+strconv.Itoa(in.LeekID), "leek "+strconv.Itoa(in.LeekID)
	}
	if in.CompositionID != 0 {
		selectors++
		path, label = "garden/get-composition-opponents/"+strconv.Itoa(in.CompositionID), "composition "+strconv.Itoa(in.CompositionID)
	}
	if in.Farmer {
		selectors++
		path, label = "garden/get-farmer-opponents", "farmer"
	}
	if selectors > 1 {
		return fail(fmt.Errorf("un seul sélecteur d'adversaires à la fois : leek_id, composition_id ou farmer"))
	}
	body, err := a.client.Get(ctx, "garden/get")
	if err != nil {
		return fail(err)
	}
	if in.Raw {
		return raw(body)
	}
	g, err := summary.Garden(body)
	if err != nil {
		return fail(err)
	}
	out := gardenResult{Garden: g}
	if path != "" {
		ops, err := a.opponents(ctx, path)
		if err != nil {
			return fail(err)
		}
		out.OpponentsFor, out.Opponents = label, ops
	}
	return ok(out)
}

// opponents lit les adversaires proposés par le matchmaking sur une route garden/get-*-opponents.
func (a *app) opponents(ctx context.Context, path string) ([]summary.Opponent, error) {
	body, err := a.client.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	return summary.Opponents(body)
}

type soloFightArgs struct {
	LeekID   int  `json:"leek_id" jsonschema:"id du poireau qui combat"`
	TargetID int  `json:"target_id,omitempty" jsonschema:"id du poireau adverse ; absent = tirage au sort parmi les adversaires proposés"`
	Wait     bool `json:"wait,omitempty" jsonschema:"attendre la génération du combat et renvoyer son résumé (comme get_fight)"`
}

func (a *app) startSoloFight(ctx context.Context, _ *mcp.CallToolRequest, in soloFightArgs) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	target, picked, err := a.target(ctx, in.TargetID, "garden/get-leek-opponents/"+strconv.Itoa(in.LeekID))
	if err != nil {
		return fail(err)
	}
	return a.launch(ctx, "garden/start-solo-fight", map[string]any{"leek_id": in.LeekID, "target_id": target}, in.Wait, picked)
}

// target renvoie la cible demandée, ou en tire une au sort parmi les adversaires
// proposés sur path quand aucune n'est donnée (picked décrit alors le tirage).
func (a *app) target(ctx context.Context, targetID int, path string) (int, *summary.Opponent, error) {
	if targetID != 0 {
		return targetID, nil, nil
	}
	ops, err := a.opponents(ctx, path)
	if err != nil {
		return 0, nil, err
	}
	if len(ops) == 0 {
		return 0, nil, fmt.Errorf("aucun adversaire proposé par le matchmaking (%s)", path)
	}
	picked := ops[rand.IntN(len(ops))]
	return picked.ID, &picked, nil
}

// launchResult est la réponse d'un lancement sans attente.
type launchResult struct {
	FightID    int    `json:"fight_id"`
	Status     int    `json:"status"`
	TargetID   int    `json:"target_id,omitempty"`
	TargetName string `json:"target_name,omitempty"`
}

// launch poste le lancement, puis renvoie soit l'id et le status du combat, soit,
// avec wait, son résumé une fois généré.
func (a *app) launch(ctx context.Context, path string, payload map[string]any, wait bool, picked *summary.Opponent) (*mcp.CallToolResult, any, error) {
	body, err := a.client.Post(ctx, path, payload)
	if err != nil {
		return fail(err)
	}
	id, err := summary.StartedFightID(body)
	if err != nil {
		return fail(err)
	}
	if wait {
		report, err := a.waitForFight(ctx, id)
		if err != nil {
			return fail(err)
		}
		items, err := a.items.Items(ctx)
		if err != nil {
			return fail(err)
		}
		s, err := summary.Fight(report, items, 0)
		if err != nil {
			return fail(err)
		}
		return ok(s)
	}
	report, err := a.fightReport(ctx, id)
	if err != nil {
		return fail(err)
	}
	status, err := summary.FightStatus(report)
	if err != nil {
		return fail(err)
	}
	out := launchResult{FightID: id, Status: status}
	if picked != nil {
		out.TargetID, out.TargetName = picked.ID, picked.Name
	}
	return ok(out)
}

// waitForFight sonde fight/get jusqu'à la fin de la génération, dans la limite
// de pollDeadline, et met le rapport terminé en cache.
func (a *app) waitForFight(ctx context.Context, id int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, a.pollDeadline)
	defer cancel()
	for {
		report, err := a.fightReport(ctx, id)
		if err != nil {
			return nil, err
		}
		status, err := summary.FightStatus(report)
		if err != nil {
			return nil, err
		}
		if status == summary.FightFinished {
			return report, nil
		}
		select {
		case <-time.After(a.pollInterval):
		case <-ctx.Done():
			return nil, fmt.Errorf("combat %d toujours en génération après %s : réessayer get_fight id=%d", id, a.pollDeadline, id)
		}
	}
}

type farmerFightArgs struct {
	TargetID int  `json:"target_id,omitempty" jsonschema:"id de l'éleveur adverse ; absent = tirage au sort parmi les éleveurs proposés"`
	Wait     bool `json:"wait,omitempty" jsonschema:"attendre la génération du combat et renvoyer son résumé (comme get_fight)"`
}

func (a *app) startFarmerFight(ctx context.Context, _ *mcp.CallToolRequest, in farmerFightArgs) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	target, picked, err := a.target(ctx, in.TargetID, "garden/get-farmer-opponents")
	if err != nil {
		return fail(err)
	}
	return a.launch(ctx, "garden/start-farmer-fight", map[string]any{"target_id": target}, in.Wait, picked)
}

type teamFightArgs struct {
	CompositionID int  `json:"composition_id" jsonschema:"id de la composition d'équipe qui combat (voir get_garden)"`
	TargetID      int  `json:"target_id,omitempty" jsonschema:"id de la composition adverse ; absent = tirage au sort parmi celles proposées"`
	Wait          bool `json:"wait,omitempty" jsonschema:"attendre la génération du combat et renvoyer son résumé (comme get_fight)"`
}

func (a *app) startTeamFight(ctx context.Context, _ *mcp.CallToolRequest, in teamFightArgs) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	target, picked, err := a.target(ctx, in.TargetID, "garden/get-composition-opponents/"+strconv.Itoa(in.CompositionID))
	if err != nil {
		return fail(err)
	}
	return a.launch(ctx, "garden/start-team-fight", map[string]any{"composition_id": in.CompositionID, "target_id": target}, in.Wait, picked)
}

type bossFightArgs struct {
	Boss         string `json:"boss" jsonschema:"id ou nom du boss (nasu_samurai, fennel_king, evil_pumpkin)"`
	Participants []int  `json:"participants,omitempty" jsonschema:"ids des poireaux engagés ; absent = tous les poireaux de l'éleveur du token"`
	Wait         bool   `json:"wait,omitempty" jsonschema:"attendre la génération du combat et renvoyer son résumé (comme get_fight)"`
}

func (a *app) startBossFight(ctx context.Context, _ *mcp.CallToolRequest, in bossFightArgs) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	bossID, err := a.resolveBoss(ctx, in.Boss)
	if err != nil {
		return fail(err)
	}
	participants := in.Participants
	if len(participants) == 0 {
		body, err := a.client.Get(ctx, "farmer/get-from-token")
		if err != nil {
			return fail(err)
		}
		if participants, err = summary.FarmerLeekIDs(body); err != nil {
			return fail(err)
		}
	}
	return a.launch(ctx, "garden/start-boss-fight", map[string]any{"boss_id": bossID, "participants": participants}, in.Wait, nil)
}

// resolveBoss accepte un id numérique ou un nom de boss/get-all (casse ignorée).
func (a *app) resolveBoss(ctx context.Context, boss string) (int, error) {
	if id, err := strconv.Atoi(strings.TrimSpace(boss)); err == nil {
		return id, nil
	}
	body, err := a.client.Get(ctx, "boss/get-all")
	if err != nil {
		return 0, err
	}
	bosses, err := summary.Bosses(body)
	if err != nil {
		return 0, err
	}
	names := make([]string, 0, len(bosses))
	for _, b := range bosses {
		if strings.EqualFold(b.Name, strings.TrimSpace(boss)) {
			return b.ID, nil
		}
		names = append(names, b.Name)
	}
	return 0, fmt.Errorf("boss %q inconnu : %s", boss, strings.Join(names, ", "))
}
