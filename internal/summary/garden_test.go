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
