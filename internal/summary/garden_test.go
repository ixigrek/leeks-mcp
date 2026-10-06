package summary

import (
	"reflect"
	"testing"
)

func TestFightStatusFinishedAndPending(t *testing.T) {
	if st, err := FightStatus(fixture(t, "fight_53988601.json")); err != nil || st != FightFinished {
		t.Fatalf("status terminé = %d, %v", st, err)
	}
	if st, err := FightStatus(fixture(t, "fight_pending.json")); err != nil || st >= FightFinished {
		t.Fatalf("status en attente = %d, %v", st, err)
	}
}

func TestStartedFightIDAcceptsKnownShapes(t *testing.T) {
	for body, want := range map[string]int{`{"fight":1}`: 1, `{"fight_id":2}`: 2, `{"fights":[3,4]}`: 3} {
		got, err := StartedFightID([]byte(body))
		if err != nil || got != want {
			t.Fatalf("%s : %d, %v", body, got, err)
		}
	}
	if _, err := StartedFightID([]byte(`{"success":true}`)); err == nil {
		t.Fatal("erreur attendue sans id")
	}
}

func TestFarmerLeekIDsSorted(t *testing.T) {
	got, err := FarmerLeekIDs(fixture(t, "farmer_token.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{135146, 135204, 135221, 135231}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ids = %v", got)
	}
}

func TestBossesListsNamesAndLevels(t *testing.T) {
	got, err := Bosses(fixture(t, "bosses.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := []BossRef{{1, "nasu_samurai", 100}, {2, "fennel_king", 200}, {3, "evil_pumpkin", 300}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("boss = %+v", got)
	}
}

func TestGardenCountersAndFlags(t *testing.T) {
	g, err := Garden(fixture(t, "garden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if g.Fights != 3040 || g.MaxFights != 3040 || g.TeamFights != 0 || !g.FarmerEnabled || g.TeamEnabled || !g.BattleRoyaleEnabled {
		t.Fatalf("garden = %+v", g)
	}
	if g.Compositions == nil || len(g.Compositions) != 0 {
		t.Fatalf("compositions = %v", g.Compositions)
	}
}

func TestOpponentsLeeks(t *testing.T) {
	ops, err := Opponents(fixture(t, "garden_leek_opponents_135146.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 5 || ops[0].ID != 83810 || ops[0].Name != "Imarmaleek" || ops[0].Level != 244 || ops[0].Talent != 1025 {
		t.Fatalf("adversaires = %+v", ops)
	}
}

func TestOpponentsFarmers(t *testing.T) {
	ops, err := Opponents(fixture(t, "garden_farmer_opponents.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 5 || ops[0].ID != 99881 || ops[0].Name != "Simon200079" || ops[0].LeekCount != 4 || ops[0].TotalLevel != 691 {
		t.Fatalf("adversaires = %+v", ops)
	}
}

func TestOpponentsCompositions(t *testing.T) {
	// Fixture écrite à la main (pas d'équipe sur le compte de test) d'après le
	// modèle Composition du client officiel.
	ops, err := Opponents(fixture(t, "garden_composition_opponents.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 2 || ops[0].ID != 9001 || ops[0].Team == nil || ops[0].Team.Name != "Potager Uni" || len(ops[0].Leeks) != 2 || ops[0].Leeks[1].Name != "Oignon2" {
		t.Fatalf("adversaires = %+v", ops)
	}
}
