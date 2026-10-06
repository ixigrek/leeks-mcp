package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ixigrek/leeks-mcp/internal/summary"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxBatch borne un lot : au-delà, l'appel dépasserait les délais d'un client MCP.
const maxBatch = 50

type batchArgs struct {
	LeekID int    `json:"leek_id" jsonschema:"poireau suivi : résultat, PV restants et VERSION sont lus pour lui (solo : celui qui combat)"`
	N      int    `json:"n" jsonschema:"nombre de combats (1 à 50)"`
	Type   string `json:"type,omitempty" jsonschema:"solo (défaut), farmer ou boss ; adversaire tiré au sort à chaque combat"`
	Boss   string `json:"boss,omitempty" jsonschema:"type boss : id ou nom du boss ; participants = tous les poireaux de l'éleveur"`
}

// batchResult est la réponse de run_batch : le bilan agrégé des combats terminés,
// et les erreurs rencontrées (lancement interrompu, combat illisible).
type batchResult struct {
	Requested int `json:"requested"`
	Launched  int `json:"launched"`
	*summary.BatchSummary
	Errors []string `json:"errors,omitempty"`
}

func (a *app) runBatch(ctx context.Context, _ *mcp.CallToolRequest, in batchArgs) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	if in.N < 1 || in.N > maxBatch {
		return fail(fmt.Errorf("n = %d : entre 1 et %d", in.N, maxBatch))
	}
	start, err := a.batchLauncher(ctx, in)
	if err != nil {
		return fail(err)
	}
	body, err := a.client.Get(ctx, "garden/get")
	if err != nil {
		return fail(err)
	}
	g, err := summary.Garden(body)
	if err != nil {
		return fail(err)
	}
	if g.Fights < in.N {
		return fail(fmt.Errorf("%d combat(s) restant(s) pour un lot de %d", g.Fights, in.N))
	}

	out := batchResult{Requested: in.N}
	var ids []int
	for i := 0; i < in.N; i++ {
		if i > 0 {
			select {
			case <-time.After(a.launchInterval):
			case <-ctx.Done():
				return fail(fmt.Errorf("lot interrompu après %d lancement(s) : %v (%w)", len(ids), ids, ctx.Err()))
			}
		}
		id, err := start(ctx)
		if err != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("lancement %d/%d : %v", i+1, in.N, err))
			break
		}
		ids = append(ids, id)
	}
	out.Launched = len(ids)
	if len(ids) == 0 {
		return fail(fmt.Errorf("aucun combat lancé : %s", out.Errors[0]))
	}

	items, err := a.items.Items(ctx)
	if err != nil {
		return fail(fmt.Errorf("combats %v lancés, mais %w", ids, err))
	}
	var outcomes []summary.Outcome
	for _, id := range ids {
		report, err := a.waitForFight(ctx, id)
		if err != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("combat %d : %v", id, err))
			continue
		}
		logs, err := a.client.Get(ctx, "fight/get-logs/"+strconv.Itoa(id))
		if err != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("logs du combat %d : %v", id, err))
			logs = nil
		}
		o, err := summary.FightOutcome(report, logs, items, in.LeekID)
		if err != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("combat %d : %v", id, err))
			continue
		}
		outcomes = append(outcomes, *o)
	}
	out.BatchSummary = summary.Batch(outcomes)
	return ok(out)
}

// batchLauncher prépare ce qui ne change pas d'un combat à l'autre (boss,
// participants) et renvoie le lancement d'un combat, qui retire l'adversaire
// au sort à chaque fois : le matchmaking renouvelle ses propositions.
func (a *app) batchLauncher(ctx context.Context, in batchArgs) (func(context.Context) (int, error), error) {
	post := func(ctx context.Context, path string, payload map[string]any) (int, error) {
		body, err := a.client.Post(ctx, path, payload)
		if err != nil {
			return 0, err
		}
		return summary.StartedFightID(body)
	}
	switch in.Type {
	case "", "solo":
		return func(ctx context.Context) (int, error) {
			target, _, err := a.target(ctx, 0, "garden/get-leek-opponents/"+strconv.Itoa(in.LeekID))
			if err != nil {
				return 0, err
			}
			return post(ctx, "garden/start-solo-fight", map[string]any{"leek_id": in.LeekID, "target_id": target})
		}, nil
	case "farmer":
		return func(ctx context.Context) (int, error) {
			target, _, err := a.target(ctx, 0, "garden/get-farmer-opponents")
			if err != nil {
				return 0, err
			}
			return post(ctx, "garden/start-farmer-fight", map[string]any{"target_id": target})
		}, nil
	case "boss":
		if in.Boss == "" {
			return nil, fmt.Errorf("type boss : préciser boss (id ou nom)")
		}
		payload, err := a.bossPayload(ctx, in.Boss, nil)
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context) (int, error) {
			return post(ctx, "garden/start-boss-fight", payload)
		}, nil
	}
	return nil, fmt.Errorf("type %q inconnu : solo, farmer ou boss", in.Type)
}
