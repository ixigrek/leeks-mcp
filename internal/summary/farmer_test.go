package summary

import "testing"

func TestFarmerPublicHasLeeksButNoInventory(t *testing.T) {
	got, err := Farmer(fixture(t, "farmer_128381.json"), items(t))
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 128381 || got.Name != "PlopVersionDeux" {
		t.Fatalf("identité : %+v", got)
	}
	if len(got.Leeks) != 4 {
		t.Fatalf("%d poireaux, attendu 4", len(got.Leeks))
	}
	var plop bool
	for _, l := range got.Leeks {
		if l.ID == 135146 && l.Name == "plop2point0" && l.Level == 233 {
			plop = true
		}
	}
	if !plop {
		t.Fatalf("plop2point0 absent : %+v", got.Leeks)
	}
	if got.Habs != nil || len(got.Inventory.Weapons) != 0 {
		t.Fatalf("données privées inattendues : %+v", got)
	}
}

func TestFarmerFromTokenHasHabsAndInventory(t *testing.T) {
	got, err := Farmer(fixture(t, "farmer_token.json"), items(t))
	if err != nil {
		t.Fatal(err)
	}
	if got.Habs == nil || *got.Habs != 11635402 || got.Crystals == nil || *got.Crystals != 5000 {
		t.Fatalf("habs/cristaux : %v / %v", got.Habs, got.Crystals)
	}
	if len(got.Inventory.Weapons) != 4 || len(got.Inventory.Chips) != 17 {
		t.Fatalf("inventaire : %d armes, %d puces", len(got.Inventory.Weapons), len(got.Inventory.Chips))
	}
	if got.Inventory.Weapons[0].Name != "pistol" || got.Inventory.Weapons[0].Count != 4 {
		t.Fatalf("première arme : %+v", got.Inventory.Weapons[0])
	}
	// L'inventaire porte l'id de chip/get-all : 1 = shock, 3 = bandage.
	if got.Inventory.Chips[0].Name != "shock" || got.Inventory.Chips[1].Name != "bandage" {
		t.Fatalf("premières puces : %+v", got.Inventory.Chips[:2])
	}
}
