package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
	"github.com/ixigrek/leeks-mcp/internal/summary"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Les loadouts (ensembles d'équipement) sont le mécanisme de l'API pour changer
// l'équipement et le capital d'un poireau avec une clé API : les routes leek/add-*,
// leek/remove-* et leek/spend-capital exigent une session de navigateur.

func (a *app) listLoadouts(ctx context.Context, _ *mcp.CallToolRequest, in rawFlag) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	body, err := a.client.Get(ctx, "loadout/get-all")
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
	s, err := summary.Loadouts(body, items)
	if err != nil {
		return fail(err)
	}
	return ok(s)
}

type saveLoadoutArgs struct {
	SetID      int            `json:"set_id,omitempty" jsonschema:"id du loadout à mettre à jour ; absent = création"`
	Name       string         `json:"name,omitempty" jsonschema:"nom du loadout (requis à la création)"`
	Icon       string         `json:"icon,omitempty" jsonschema:"icône : nom de caractéristique (strength, agility…) ou emoji ; défaut strength"`
	FromLeekID int            `json:"from_leek_id,omitempty" jsonschema:"préremplir armes, puces, composants et capital depuis le build actuel de ce poireau (sinon, à la mise à jour, depuis le loadout existant)"`
	Weapons    []string       `json:"weapons,omitempty" jsonschema:"armes (noms anglais de l'API, ex. laser) ; remplace la liste préremplie"`
	Chips      []string       `json:"chips,omitempty" jsonschema:"puces (ex. flash) ; remplace la liste préremplie"`
	Stats      map[string]int `json:"stats,omitempty" jsonschema:"capital total investi par caractéristique (life, strength, wisdom, agility, resistance, science, magic, frequency, cores, ram, tp, mp) ; fusionné stat par stat avec le préremplissage, 0 pour retirer"`
}

func (a *app) saveLoadout(ctx context.Context, _ *mcp.CallToolRequest, in saveLoadoutArgs) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	items, err := a.items.Items(ctx)
	if err != nil {
		return fail(err)
	}
	// Point de départ : le loadout existant (mise à jour), puis le build du poireau.
	base := summary.Loadout{Weapons: []int{}, Chips: []int{}, Components: []summary.LoadoutComponent{}, Stats: map[string]int{}}
	if in.SetID != 0 {
		body, err := a.client.Get(ctx, "loadout/get-all")
		if err != nil {
			return fail(err)
		}
		existing, err := summary.FindLoadout(body, in.SetID)
		if err != nil {
			return fail(err)
		}
		base = *existing
		base.Weapons = append(append([]int{}, base.Weapons...), base.ForgottenWeapons...)
		if base.Stats == nil {
			base.Stats = map[string]int{}
		}
	}
	var level int
	if in.FromLeekID != 0 {
		build, err := a.build(ctx, in.FromLeekID)
		if err != nil {
			return fail(err)
		}
		level = build.Level
		base.Weapons, base.Chips, base.Components = build.Weapons, build.Chips, build.Components
		base.Stats = summary.CapitalSpent(build.Level, build.Stats)
	}
	if in.Name != "" {
		base.Name = in.Name
	}
	if base.Name == "" {
		return fail(fmt.Errorf("name requis à la création"))
	}
	if in.Icon != "" {
		base.Icon = in.Icon
	}
	if base.Icon == "" {
		base.Icon = "strength"
	}
	if in.Weapons != nil {
		if base.Weapons, err = resolveWeapons(items, in.Weapons); err != nil {
			return fail(err)
		}
	}
	if in.Chips != nil {
		if base.Chips, err = resolveChips(items, in.Chips); err != nil {
			return fail(err)
		}
	}
	for stat, capital := range in.Stats {
		if !summary.IsStat(stat) {
			return fail(fmt.Errorf("caractéristique %q inconnue", stat))
		}
		if capital < 0 {
			return fail(fmt.Errorf("%s : capital négatif", stat))
		}
		if capital == 0 {
			delete(base.Stats, stat)
		} else {
			base.Stats[stat] = capital
		}
	}
	if level != 0 {
		total := 0
		for _, c := range base.Stats {
			total += c
		}
		if max := summary.TotalCapital(level); total > max {
			return fail(fmt.Errorf("%d capital demandé, %d au total au niveau %d", total, max, level))
		}
	}
	// L'API attend les listes et les stats en chaînes JSON, comme le client officiel.
	payload := map[string]any{
		"name": base.Name, "icon": base.Icon,
		"weapons": jsonString(base.Weapons), "chips": jsonString(base.Chips),
		"components": jsonString(base.Components), "stats": jsonString(base.Stats),
	}
	var body []byte
	if in.SetID != 0 {
		payload["set_id"] = in.SetID
		body, err = a.client.Put(ctx, "loadout/update", payload)
	} else {
		body, err = a.client.Post(ctx, "loadout/create", payload)
	}
	if err != nil {
		return fail(err)
	}
	set, err := summary.LoadoutSet(body)
	if err != nil {
		return fail(err)
	}
	return ok(set.Summarize(items))
}

func jsonString(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

type applyLoadoutArgs struct {
	SetID     int  `json:"set_id" jsonschema:"id du loadout (voir list_loadouts)"`
	LeekID    int  `json:"leek_id" jsonschema:"id du poireau"`
	UseRestat bool `json:"use_restat,omitempty" jsonschema:"consommer une potion de restat si le loadout réduit le capital d'une stat (sinon seul l'ajout de capital est possible)"`
}

func (a *app) applyLoadout(ctx context.Context, _ *mcp.CallToolRequest, in applyLoadoutArgs) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	body, err := a.client.Post(ctx, "loadout/apply", map[string]any{"set_id": in.SetID, "leek_id": in.LeekID, "use_restat": in.UseRestat})
	if err != nil {
		return fail(err)
	}
	items, err := a.items.Items(ctx)
	if err != nil {
		return fail(err)
	}
	s, err := summary.Applied(body, in.LeekID, items)
	if err != nil {
		return fail(err)
	}
	return ok(s)
}

type deleteLoadoutArgs struct {
	SetID int `json:"set_id" jsonschema:"id du loadout à supprimer"`
}

func (a *app) deleteLoadout(ctx context.Context, _ *mcp.CallToolRequest, in deleteLoadoutArgs) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	if _, err := a.client.Delete(ctx, "loadout/delete", map[string]any{"set_id": in.SetID}); err != nil {
		return fail(err)
	}
	return ok(map[string]any{"deleted": in.SetID})
}

// build lit l'état privé d'un poireau.
func (a *app) build(ctx context.Context, leekID int) (*summary.LeekBuild, error) {
	body, err := a.client.Get(ctx, "leek/get-private/"+strconv.Itoa(leekID))
	if err != nil {
		return nil, err
	}
	return summary.Build(body)
}

// resolveWeapons traduit des noms ou ids d'armes en templates (item), sans doublon.
func resolveWeapons(items *leekwars.Items, names []string) ([]int, error) {
	out := []int{}
	for _, name := range names {
		var found []*leekwars.Weapon
		for _, m := range items.Find(name) {
			if m.Weapon != nil {
				found = append(found, m.Weapon)
			}
		}
		switch len(found) {
		case 0:
			return nil, fmt.Errorf("arme %q inconnue", name)
		case 1:
		default:
			return nil, fmt.Errorf("arme %q ambiguë", name)
		}
		if contains(out, found[0].Item) {
			return nil, fmt.Errorf("arme %s en double", found[0].Name)
		}
		out = append(out, found[0].Item)
	}
	return out, nil
}

// resolveChips traduit des noms ou ids de puces en ids de chip/get-all (ceux des loadouts), sans doublon.
func resolveChips(items *leekwars.Items, names []string) ([]int, error) {
	out := []int{}
	for _, name := range names {
		var chip *leekwars.Chip
		for _, m := range items.Find(name) {
			if m.Chip != nil {
				chip = m.Chip
				break
			}
		}
		if chip == nil {
			return nil, fmt.Errorf("puce %q inconnue", name)
		}
		if contains(out, chip.ID) {
			return nil, fmt.Errorf("puce %s en double", chip.Name)
		}
		out = append(out, chip.ID)
	}
	return out, nil
}

func contains(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
