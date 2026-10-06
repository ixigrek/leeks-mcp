package summary

import (
	"strings"
	"testing"
)

func fightSummary(t *testing.T) *FightSummary {
	t.Helper()
	got, err := Fight(fixture(t, "fight_53988601.json"), items(t), 0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestFightHeader(t *testing.T) {
	got := fightSummary(t)
	if got.ID != 53988601 || got.Winner != 1 || got.Duration != 41 || got.Date == "" {
		t.Fatalf("en-tête : %+v", got)
	}
	if len(got.Entities) != 32 {
		t.Fatalf("%d entités, attendu 32", len(got.Entities))
	}
	var plop *EntitySummary
	for i := range got.Entities {
		if got.Entities[i].Name == "plop2point0" {
			plop = &got.Entities[i]
		}
	}
	if plop == nil || plop.ID != 15 || plop.LeekID != 135146 || plop.Team != 16 || plop.Life != 1736 {
		t.Fatalf("plop2point0 : %+v", plop)
	}
}

func TestFightTurnsCoverWholeFight(t *testing.T) {
	got := fightSummary(t)
	if len(got.Turns) != 41 || got.Turns[0].Turn != 1 || got.Turns[40].Turn != 41 {
		t.Fatalf("%d tours, premier %d, dernier %d", len(got.Turns), got.Turns[0].Turn, got.Turns[len(got.Turns)-1].Turn)
	}
}

func TestFightTurnOneActionsOfPlop(t *testing.T) {
	got := fightSummary(t)
	var e *EntityTurn
	for i := range got.Turns[0].Entities {
		if got.Turns[0].Entities[i].ID == 15 {
			e = &got.Turns[0].Entities[i]
		}
	}
	if e == nil {
		t.Fatal("entité 15 absente du tour 1")
	}
	// Actions 37-44 du rapport : LEEK_TURN 15, steroid sur 598, SET_WEAPON rhino, tir sur 599 (-526 à l'entité 1), déplacement vers 508 en 5 PM.
	if len(e.Chips) != 1 || e.Chips[0].Chip != "steroid" || e.Chips[0].Cell != 598 {
		t.Fatalf("puces : %+v", e.Chips)
	}
	if len(e.WeaponShots) != 1 || e.WeaponShots[0].Weapon != "rhino" || e.WeaponShots[0].Cell != 599 {
		t.Fatalf("tirs : %+v", e.WeaponShots)
	}
	if e.DamageDealt != 526 || e.DamageTaken != 0 {
		t.Fatalf("dégâts infligés %d / reçus %d", e.DamageDealt, e.DamageTaken)
	}
	if len(e.Moves) != 1 || e.Moves[0].To != 508 || e.Moves[0].MP != 5 {
		t.Fatalf("déplacements : %+v", e.Moves)
	}
	if e.LifeEnd != 1736 {
		t.Fatalf("PV fin de tour %d", e.LifeEnd)
	}
}

func TestFightLifeFollowsDamage(t *testing.T) {
	got := fightSummary(t)
	// Entité 1 (Grubert, 1567 PV) perd 526 au tour 1 par le tir de plop2point0, puis se soigne de 369.
	for _, e := range got.Turns[0].Entities {
		if e.ID == 1 {
			if e.DamageTaken != 526 || e.Heal != 369 || e.LifeEnd != 1567-526+369 {
				t.Fatalf("entité 1 : %+v", e)
			}
			return
		}
	}
	t.Fatal("entité 1 absente du tour 1")
}

func TestFightDeathsAndUnknownActions(t *testing.T) {
	got := fightSummary(t)
	if len(got.Deaths) != 29 {
		t.Fatalf("%d morts, attendu 29", len(got.Deaths))
	}
	// Première mort à l'index 292, après le NEW_TURN 2 (index 267).
	if got.Deaths[0].Entity != 19 || got.Deaths[0].Turn != 2 {
		t.Fatalf("première mort : %+v", got.Deaths[0])
	}
	if len(got.UnknownActions) != 0 {
		t.Fatalf("actions inconnues : %v", got.UnknownActions)
	}
}

func TestFightFilteredByLeekKeepsOnlyThatEntity(t *testing.T) {
	got, err := Fight(fixture(t, "fight_53988601.json"), items(t), 135146)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Turns) != 41 || len(got.Entities) != 32 {
		t.Fatalf("%d tours, %d entités", len(got.Turns), len(got.Entities))
	}
	for _, turn := range got.Turns {
		for _, e := range turn.Entities {
			if e.ID != 15 {
				t.Fatalf("tour %d : entité %d présente malgré le filtre", turn.Turn, e.ID)
			}
		}
	}
	if len(got.Turns[0].Entities) != 1 || got.Turns[0].Entities[0].DamageDealt != 526 {
		t.Fatalf("tour 1 : %+v", got.Turns[0].Entities)
	}
}

func TestFightFilteredByUnknownLeekFails(t *testing.T) {
	if _, err := Fight(fixture(t, "fight_53988601.json"), items(t), 999); err == nil {
		t.Fatal("erreur attendue pour un poireau absent du combat")
	}
}

// Les USE_CHIP portent le template de la puce, qui diffère de son id dans
// chip/get-all : 38 = armoring, 40 = puny_bulb, 1 = bandage (et non shock).
func TestFightChipNamesUseTemplate(t *testing.T) {
	got, err := Fight(fixture(t, "fight_53994496.json"), items(t), 0)
	if err != nil {
		t.Fatal(err)
	}
	var e *EntityTurn
	for i := range got.Turns[0].Entities {
		if got.Turns[0].Entities[i].ID == 1 {
			e = &got.Turns[0].Entities[i]
		}
	}
	if e == nil {
		t.Fatal("entité 1 absente du tour 1")
	}
	var names []string
	for _, c := range e.Chips {
		names = append(names, c.Chip)
	}
	if want := "armoring puny_bulb bandage"; strings.Join(names, " ") != want {
		t.Fatalf("puces du tour 1 : %v, attendu %s", names, want)
	}
}
