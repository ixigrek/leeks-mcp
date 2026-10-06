package summary

import (
	"reflect"
	"strings"
	"testing"
)

func outcome(t *testing.T, id string) Outcome {
	t.Helper()
	o, err := FightOutcome(fixture(t, "fight_"+id+".json"), fixture(t, "logs_"+id+".json"), items(t), 135146)
	if err != nil {
		t.Fatal(err)
	}
	return *o
}

func TestFightOutcome(t *testing.T) {
	cases := []struct {
		id   string
		want Outcome
	}{
		// Boss : le _nasu puis le _grp affichent leur VERSION au tour 1.
		{"53996480", Outcome{Fight: 53996480, Result: ResultWin, Turns: 17, Life: 1587, Versions: []string{"N-2026-10-06d", "G00-2026-10-06h"}}},
		// Solo perdu : « T1 v2026-10-06d budget… », mais pas « vers 56 ».
		{"53994496", Outcome{Fight: 53994496, Result: ResultDefeat, Turns: 12, Life: 0, Versions: []string{"2026-10-06d"}}},
		{"53996476", Outcome{Fight: 53996476, Result: ResultDraw, Turns: 64, Life: 1731, Versions: []string{"N-2026-10-06d", "G00-2026-10-06h"}}},
	}
	for _, c := range cases {
		if got := outcome(t, c.id); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s : %+v, attendu %+v", c.id, got, c.want)
		}
	}
}

func TestFightOutcomeWithoutLogs(t *testing.T) {
	o, err := FightOutcome(fixture(t, "fight_53994496.json"), nil, items(t), 135146)
	if err != nil || o.Result != ResultDefeat || len(o.Versions) != 0 {
		t.Fatalf("%+v %v", o, err)
	}
}

func TestFightOutcomeLeekAbsent(t *testing.T) {
	_, err := FightOutcome(fixture(t, "fight_53994496.json"), nil, items(t), 1)
	if err == nil || !strings.Contains(err.Error(), "ne participe pas") {
		t.Fatalf("erreur attendue : %v", err)
	}
}

func TestBatch(t *testing.T) {
	b := Batch([]Outcome{outcome(t, "53996480"), outcome(t, "53994496"), outcome(t, "53996476"), outcome(t, "53996480")})
	want := &BatchSummary{
		Fights: 4, Wins: 2, Draws: 1, Defeats: 1,
		AvgTurns: 27.5, AvgLife: 1226.3, AvgLifeWins: 1587,
		Versions:  map[string]int{"N-2026-10-06d + G00-2026-10-06h": 3, "2026-10-06d": 1},
		DefeatIDs: []int{53994496},
	}
	if !reflect.DeepEqual(b, want) {
		t.Fatalf("%+v\nattendu %+v", b, want)
	}
	empty := Batch(nil)
	if empty.Fights != 0 || empty.AvgTurns != 0 || empty.DefeatIDs == nil || empty.Versions == nil {
		t.Fatalf("lot vide : %+v", empty)
	}
	if v := Batch([]Outcome{{Result: ResultWin}}).Versions; v[unknownVersion] != 1 {
		t.Fatalf("VERSION absente : %v", v)
	}
}

func TestFightEntityLifeEnd(t *testing.T) {
	s, err := Fight(fixture(t, "fight_53994496.json"), items(t), 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range s.Entities {
		if e.LeekID == 135146 && e.LifeEnd != 0 {
			t.Fatalf("poireau mort : life_end %d", e.LifeEnd)
		}
	}
}
