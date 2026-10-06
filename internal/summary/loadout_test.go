package summary

import (
	"strings"
	"testing"
)

func TestLoadoutsResolvesNames(t *testing.T) {
	got, err := Loadouts(fixture(t, "loadouts.json"), items(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Loadouts) != 1 {
		t.Fatalf("%d loadouts", len(got.Loadouts))
	}
	l := got.Loadouts[0]
	if l.ID != 1485 || l.Name != "mcp-test" || l.Icon != "strength" || l.Components != 6 || l.Capital != 1370 {
		t.Fatalf("loadout : %+v", l)
	}
	if strings.Join(l.Weapons, ",") != "laser,grenade_launcher,rhino,"+weaponName(180, items(t)) || l.Chips[0] != "rockfall" || l.Stats["strength"] != 700 {
		t.Fatalf("équipement : %+v", l)
	}
	if len(got.OwnedWeapons) != 13 || got.OwnedWeapons[0] != "pistol" || len(got.OwnedChips) != 41 {
		t.Fatalf("possédés : %v %v", got.OwnedWeapons, got.OwnedChips)
	}
	empty, err := Loadouts(fixture(t, "loadouts_empty.json"), items(t))
	if err != nil || len(empty.Loadouts) != 0 {
		t.Fatalf("vide : %+v %v", empty, err)
	}
}

func TestLoadoutSetAndFind(t *testing.T) {
	set, err := LoadoutSet(fixture(t, "loadout_update.json"))
	if err != nil || set.ID != 1485 || set.Name != "mcp-test-2" || set.Icon != "agility" || len(set.Weapons) != 4 || len(set.Components) != 6 {
		t.Fatalf("set : %+v %v", set, err)
	}
	if string(set.Components[0].Stats) != "null" {
		t.Fatalf("stats de composant : %s", set.Components[0].Stats)
	}
	if _, err := LoadoutSet(fixture(t, "loadout_delete.json")); err == nil {
		t.Fatal("set attendu absent")
	}
	found, err := FindLoadout(fixture(t, "loadouts.json"), 1485)
	if err != nil || found.Name != "mcp-test" {
		t.Fatalf("find : %+v %v", found, err)
	}
	if _, err := FindLoadout(fixture(t, "loadouts.json"), 1); err == nil {
		t.Fatal("loadout 1 trouvé")
	}
}

func TestAppliedSummarizesLeek(t *testing.T) {
	got, err := Applied(fixture(t, "loadout_apply.json"), 135146, items(t))
	if err != nil {
		t.Fatal(err)
	}
	if got.LeekID != 135146 || len(got.Weapons) != 4 || got.Weapons[0] != "laser" || len(got.Chips) != 12 || got.Components != 6 {
		t.Fatalf("appliqué : %+v", got)
	}
	if got.StatsChanged || got.RestatUsed || len(got.Skipped) != 0 {
		t.Fatalf("drapeaux : %+v", got)
	}
}
