package summary

import "testing"

func TestItemFromWeaponKeepsBothIdentifiers(t *testing.T) {
	it := items(t)
	got := Item(it.Find("laser")[0])
	if got.Kind != "weapon" || got.ID != 6 || got.Item != 42 || got.Name != "laser" {
		t.Fatalf("Item(laser) = %+v", got)
	}
	if got.Cost == 0 || got.MaxRange == 0 || len(got.Effects) == 0 {
		t.Fatalf("champs manquants : %+v", got)
	}
}

func TestItemFromChipHasCooldownAndNoItem(t *testing.T) {
	it := items(t)
	got := Item(it.Find("leather_boots")[0])
	// id comme dans get_leek (14), template des rapports à part (30).
	if got.Kind != "chip" || got.ID != 14 || got.Template != 30 || got.Item != 0 || got.Cooldown == nil {
		t.Fatalf("Item(leather_boots) = %+v", got)
	}
}
