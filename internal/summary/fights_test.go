package summary

import "testing"

func TestFightListDefaultsToTenMostRecent(t *testing.T) {
	got, err := FightList(fixture(t, "leek_135146.json"), 135146, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Fatalf("%d combats, attendu 10", len(got))
	}
	if got[0].ID != 53988601 || got[0].Duration != 41 {
		t.Fatalf("premier : %+v", got[0])
	}
}

func TestFightListFiltersByResult(t *testing.T) {
	got, err := FightList(fixture(t, "leek_135146.json"), 135146, "defeat", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("aucune défaite")
	}
	for _, f := range got {
		if f.Result != "defeat" {
			t.Fatalf("résultat %q dans le filtre defeat", f.Result)
		}
	}
}

func TestFightListOpponentsExcludeOwnTeam(t *testing.T) {
	got, err := FightList(fixture(t, "leek_135146.json"), 135146, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	// Combat 53988601 : plop2point0 dans leeks1 (17 poireaux), coffres -100 et -101 en face.
	if len(got[0].Opponents) != 2 {
		t.Fatalf("adversaires : %v", got[0].Opponents)
	}
	for _, id := range got[0].Opponents {
		if id == 135146 {
			t.Fatal("le poireau est dans ses propres adversaires")
		}
	}
}

func TestFightListRejectsUnknownResult(t *testing.T) {
	if _, err := FightList(fixture(t, "leek_135146.json"), 135146, "loss", 0); err == nil {
		t.Fatal("erreur attendue pour result=loss")
	}
}
