package summary

import "testing"

func TestLogsGroupedByTurnWithEntity(t *testing.T) {
	got, err := Logs(fixture(t, "logs_53988601.json"), fixture(t, "fight_53988601.json"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Fight != 53988601 {
		t.Fatalf("fight = %d", got.Fight)
	}
	if len(got.Turns) < 3 || got.Turns[0].Turn != 1 || got.Turns[1].Turn != 2 || got.Turns[2].Turn != 3 {
		t.Fatalf("tours : %+v", got.Turns)
	}
	// Index 37, 39 et 43 du rapport = tour 1 : 4 + 1 + 1 lignes, entité 15, IA « force ».
	t1 := got.Turns[0]
	if len(t1.Entries) != 6 {
		t.Fatalf("%d lignes au tour 1, attendu 6", len(t1.Entries))
	}
	first := t1.Entries[0]
	if first.Entity != 15 || first.LeekID != 135146 || first.AI != "force" || first.Line != 131 {
		t.Fatalf("première ligne : %+v", first)
	}
	if len(first.Text) < 10 || first.Text[:8] != "profil :" {
		t.Fatalf("texte : %q", first.Text)
	}
}

func TestLogsFilterByLeekID(t *testing.T) {
	got, err := Logs(fixture(t, "logs_53988601.json"), fixture(t, "fight_53988601.json"), 135146)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Turns) == 0 {
		t.Fatal("aucune ligne pour 135146")
	}
	other, err := Logs(fixture(t, "logs_53988601.json"), fixture(t, "fight_53988601.json"), 127085)
	if err != nil {
		t.Fatal(err)
	}
	if len(other.Turns) != 0 {
		t.Fatalf("lignes inattendues pour 127085 : %+v", other.Turns)
	}
}

func TestLogsKeepTurns(t *testing.T) {
	s, err := Logs(fixture(t, "logs_53994496.json"), fixture(t, "fight_53994496.json"), 0)
	if err != nil {
		t.Fatal(err)
	}
	s.KeepTurns(3, 5)
	if len(s.Turns) == 0 {
		t.Fatal("aucun tour entre 3 et 5")
	}
	for _, turn := range s.Turns {
		if turn.Turn < 3 || turn.Turn > 5 {
			t.Fatalf("tour %d hors de 3-5", turn.Turn)
		}
	}
}

// Combat sans logs (53996176) : l'API renvoie [] et non {}.
func TestLogsEmptyArray(t *testing.T) {
	s, err := Logs(fixture(t, "logs_empty.json"), fixture(t, "fight_53994496.json"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Turns) != 0 || s.Turns == nil {
		t.Fatalf("tours : %+v", s.Turns)
	}
}
