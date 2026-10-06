package leekwars

import (
	"os"
	"strings"
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

// Find par nombre renvoie tous les objets dont l'id ou l'item (arme) ou l'id (puce)
// vaut ce nombre, avec le champ qui a correspondu ; le template d'une puce n'est
// pas consulté s'il y a déjà un résultat (14 = leather_boots, pas rockfall).
func TestFindByNumericIDListsEveryItem(t *testing.T) {
	items := loadItems(t)
	for q, want := range map[string]string{
		"174": "chip:manumission:id",
		"14":  "weapon:katana:id chip:leather_boots:id",
		"25":  "weapon:lightninger:id chip:steroid:id",
		"42":  "weapon:sun_spear:id weapon:laser:item",
	} {
		var got []string
		for _, m := range items.Find(q) {
			name := ""
			if m.Weapon != nil {
				name = m.Weapon.Name
			} else {
				name = m.Chip.Name
			}
			got = append(got, m.Kind+":"+name+":"+m.By)
		}
		if strings.Join(got, " ") != want {
			t.Fatalf("Find(%s) = %v, attendu %s", q, got, want)
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

// Sans objet de cet identifiant, Find se rabat sur les identifiants de rapport.
func TestFindByNumericIDFallsBackToReportIDs(t *testing.T) {
	items := loadItems(t)
	got := items.Find("61")
	if len(got) != 1 || got[0].Chip == nil || got[0].Chip.Name != "venom" || got[0].By != "template" {
		t.Fatalf("Find(61) = %+v", got)
	}
}

func TestFindUnknownReturnsNothing(t *testing.T) {
	items := loadItems(t)
	if got := items.Find("objet_imaginaire"); len(got) != 0 {
		t.Fatalf("Find inattendu : %+v", got)
	}
}
