package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ixigrek/leeks-mcp/internal/summary"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type scoutArgs struct {
	LeekID int `json:"leek_id" jsonschema:"id du poireau adverse"`
	Limit  int `json:"limit,omitempty" jsonschema:"nombre maximal de combats contre nous listés (défaut 10)"`
}

func (a *app) scout(ctx context.Context, _ *mcp.CallToolRequest, in scoutArgs) (*mcp.CallToolResult, any, error) {
	public, err := a.client.Get(ctx, "leek/get/"+strconv.Itoa(in.LeekID))
	if err != nil {
		return fail(err)
	}
	items, err := a.items.Items(ctx)
	if err != nil {
		return fail(err)
	}
	s, err := summary.Scout(public, items)
	if err != nil {
		return fail(err)
	}
	if !a.client.HasToken() {
		s.Notes = append(s.Notes, "vs_us omis : token requis pour savoir qui est « nous »")
		return ok(s)
	}
	v, err := a.vsUs(ctx, in.LeekID, in.Limit)
	if err != nil {
		s.Notes = append(s.Notes, "vs_us omis : "+err.Error())
	} else {
		s.VsUs = v
	}
	return ok(s)
}

// vsUs lit l'historique de l'adversaire et n'en garde que les combats contre
// l'éleveur du token ou l'un de ses poireaux.
func (a *app) vsUs(ctx context.Context, leekID, limit int) (*summary.VsUs, error) {
	me, err := a.client.Get(ctx, "farmer/get-from-token")
	if err != nil {
		return nil, err
	}
	var farmer struct {
		Farmer struct {
			ID int `json:"id"`
		} `json:"farmer"`
	}
	if err := json.Unmarshal(me, &farmer); err != nil {
		return nil, fmt.Errorf("farmer/get-from-token : %w", err)
	}
	leeks, err := summary.FarmerLeekIDs(me)
	if err != nil {
		return nil, err
	}
	history, err := a.client.Get(ctx, "history/get-leek-history/"+strconv.Itoa(leekID))
	if err != nil {
		return nil, err
	}
	return summary.ScoutVsUs(history, leekID, farmer.Farmer.ID, leeks, limit)
}
