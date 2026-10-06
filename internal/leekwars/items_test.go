package leekwars

import (
	"os"
	"testing"
)

func loadItems(t *testing.T) *Items {
	t.Helper()
	weapons, err := os.ReadFile("../../testdata/weapons.json")
	if err != nil {
		t.Fatal(err)
	}
	chips, err := os.ReadFile("../../testdata/chips.json")
	if err != nil {
		t.Fatal(err)
	}
	items, err := ParseItems(weapons, chips)
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func TestWeaponByIDAndByItem(t *testing.T) {
	items := loadItems(t)
	byID := items.WeaponByID(6)
	if byID == nil || byID.Name != "laser" {
		t.Fatalf("WeaponByID(6) = %+v", byID)
	}
	byItem := items.WeaponByItem(42)
	if byItem == nil || byItem.Name != "laser" {
		t.Fatalf("WeaponByItem(42) = %+v", byItem)
	}
	if items.WeaponByID(999999) != nil {
		t.Fatal("arme inconnue attendue nil")
	}
}

func TestChipByID(t *testing.T) {
	items := loadItems(t)
	// L'identifiant partout utilisé (rapports, leek/get, loadouts) est le template,
	// pas le champ id de chip/get-all : 14 = rockfall (id 32), 16 = stalactite (id 30).
	for id, name := range map[int]string{14: "rockfall", 16: "stalactite", 30: "leather_boots"} {
		c := items.ChipByID(id)
		if c == nil || c.Name != name {
			t.Fatalf("ChipByID(%d) = %+v, attendu %s", id, c, name)
		}
	}
}

func TestFindByNameIgnoresCaseAndSeparators(t *testing.T) {
	items := loadItems(t)
	for _, q := range []string{"laser", "Laser", "LASER"} {
		got := items.Find(q)
		if len(got) != 1 || got[0].Kind != "weapon" || got[0].Weapon.Name != "laser" {
			t.Fatalf("Find(%q) = %+v", q, got)
		}
	}
	got := items.Find("sun spear")
	if len(got) != 1 || got[0].Weapon == nil || got[0].Weapon.Name != "sun_spear" {
		t.Fatalf("Find(sun spear) = %+v", got)
	}
}

func TestFindByNumericIDReturnsWeaponsAndChips(t *testing.T) {
	items := loadItems(t)
	// 42 est à la fois l'item du laser et l'id de l'arme sun_spear et peut-être une puce.
	got := items.Find("42")
	names := map[string]bool{}
	for _, m := range got {
		if m.Weapon != nil {
			names[m.Weapon.Name] = true
		}
		if m.Chip != nil {
			names["chip:"+m.Chip.Name] = true
		}
	}
	if !names["laser"] || !names["sun_spear"] {
		t.Fatalf("Find(42) = %v", names)
	}
}

func TestFindUnknownReturnsNothing(t *testing.T) {
	items := loadItems(t)
	if got := items.Find("objet_imaginaire"); len(got) != 0 {
		t.Fatalf("Find inattendu : %+v", got)
	}
}
