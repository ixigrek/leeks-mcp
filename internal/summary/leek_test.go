package summary

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestLeekResolvesEquipmentAndRecord(t *testing.T) {
	got, err := Leek(fixture(t, "leek_135146.json"), nil, items(t))
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 135146 || got.Name != "plop2point0" || got.Level != 233 {
		t.Fatalf("identité : %+v", got)
	}
	if got.Stats["strength"] != 600 || got.TotalStats["strength"] < 600 {
		t.Fatalf("stats : %v / %v", got.Stats, got.TotalStats)
	}
	var laser bool
	for _, w := range got.Weapons {
		if w.Name == "laser" && w.ID == 6 && w.Item == 42 {
			laser = true
		}
	}
	if !laser {
		t.Fatalf("laser absent : %+v", got.Weapons)
	}
	// leek/get → chips[].template est l'id de chip/get-all : 14 = leather_boots.
	var boots bool
	for _, c := range got.Chips {
		if c.Name == "leather_boots" && c.ID == 14 {
			boots = true
		}
	}
	if !boots {
		t.Fatalf("leather_boots absente : %+v", got.Chips)
	}
	if got.AI.Name != "force" {
		t.Fatalf("ai = %+v", got.AI)
	}
	if got.Record.Victories == 0 && got.Record.Defeats == 0 {
		t.Fatalf("record vide : %+v", got.Record)
	}
	if len(got.Fights) != 10 {
		t.Fatalf("%d combats, attendu 10", len(got.Fights))
	}
	if got.Fights[0].ID != 53988601 || got.Fights[0].Result != "defeat" || got.Fights[0].Date == "" {
		t.Fatalf("premier combat : %+v", got.Fights[0])
	}
	if got.Capital != nil {
		t.Fatal("capital attendu absent sans get-private")
	}
}

func TestLeekMergesPrivateData(t *testing.T) {
	got, err := Leek(fixture(t, "leek_135146.json"), fixture(t, "leek_private_135146.json"), items(t))
	if err != nil {
		t.Fatal(err)
	}
	if got.Capital == nil || *got.Capital != 25 {
		t.Fatalf("capital = %v", got.Capital)
	}
	if got.Spent["strength"] != 700 || got.Spent["tp"] != 255 || len(got.Spent) != 5 {
		t.Fatalf("capital dépensé = %v", got.Spent)
	}
	if len(got.Components) != 6 {
		t.Fatalf("%d composants, attendu 6 (les emplacements vides sont ignorés)", len(got.Components))
	}
}

// Un poireau sans défaite a "ratio": "∞" ; le résumé le garde tel quel.
func TestLeekInfiniteRatio(t *testing.T) {
	got, err := Leek(fixture(t, "leek_135231.json"), nil, items(t))
	if err != nil {
		t.Fatal(err)
	}
	if got.Record.Victories != 2 || got.Record.Defeats != 0 || !math.IsInf(float64(got.Record.Ratio), 1) {
		t.Fatalf("record : %+v", got.Record)
	}
	out, err := json.Marshal(got.Record)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"ratio":"∞"`) {
		t.Fatalf("ratio sérialisé : %s", out)
	}
	if len(got.Chips) != 6 || got.Chips[3].ID != 5 || got.Chips[3].Name != "flame" {
		t.Fatalf("puces : %+v", got.Chips)
	}
}

func TestLeekAddItemDetails(t *testing.T) {
	it := items(t)
	s, err := Leek(fixture(t, "leek_135146.json"), nil, it)
	if err != nil {
		t.Fatal(err)
	}
	if s.Chips[0].Details != nil {
		t.Fatal("fiches jointes sans demande")
	}
	s.AddItemDetails(it)
	for _, c := range s.Chips {
		if c.Details == nil || c.Details.ID != c.ID || c.Details.Name != c.Name || len(c.Details.Effects) == 0 {
			t.Fatalf("puce %+v : fiche %+v", c, c.Details)
		}
	}
	for _, w := range s.Weapons {
		if w.Details == nil || w.Details.Item != w.Item || w.Details.Cost == 0 {
			t.Fatalf("arme %+v : fiche %+v", w, w.Details)
		}
	}
}
