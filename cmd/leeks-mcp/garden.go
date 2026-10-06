package main

import (
	"context"
	"fmt"
	"strconv"

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
