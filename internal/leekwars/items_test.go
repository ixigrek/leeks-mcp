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

// Une puce a deux identifiants : id (leek/get, inventaire, loadouts) et
// template (USE_CHIP des rapports). 14 = leather_boots (template 30) mais
// template 14 = rockfall (id 32).
func TestChipByIDAndByTemplate(t *testing.T) {
	items := loadItems(t)
	for id, name := range map[int]string{14: "leather_boots", 11: "vaccine", 35: "regeneration", 174: "manumission", 155: "knowledge"} {
		c := items.ChipByID(id)
		if c == nil || c.Name != name {
			t.Fatalf("ChipByID(%d) = %+v, attendu %s", id, c, name)
		}
	}
	for tpl, name := range map[int]string{14: "rockfall", 16: "stalactite", 30: "leather_boots", 1: "bandage"} {
		c := items.ChipByTemplate(tpl)
		if c == nil || c.Name != name {
			t.Fatalf("ChipByTemplate(%d) = %+v, attendu %s", tpl, c, name)
		}
	}
	if items.ChipName(174) != "manumission" || items.ChipNameByTemplate(100) != "manumission" {
		t.Fatalf("noms : %s / %s", items.ChipName(174), items.ChipNameByTemplate(100))
	}
	if items.ChipName(999999) != "chip_999999" || items.ChipNameByTemplate(999999) != "chip_999999" {
		t.Fatal("puce inconnue : chip_<id> attendu")
	}
}

// Find par nombre cherche une puce par id puis par template, comme pour les armes.
func TestFindByNumericIDMatchesChipIDAndTemplate(t *testing.T) {
	items := loadItems(t)
	got := items.Find("174")
	if len(got) != 1 || got[0].Chip == nil || got[0].Chip.Name != "manumission" {
		t.Fatalf("Find(174) = %+v", got)
	}
	names := map[string]bool{}
	for _, m := range items.Find("14") {
		if m.Chip != nil {
			names[m.Chip.Name] = true
		}
	}
	if !names["leather_boots"] || !names["rockfall"] {
		t.Fatalf("Find(14) = %v, attendu leather_boots et rockfall", names)
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
