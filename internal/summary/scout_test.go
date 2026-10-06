package summary

import (
	"testing"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
)

func TestScoutEffectsScaledByStats(t *testing.T) {
	s, err := Scout(fixture(t, "leek_135146.json"), items(t))
	if err != nil {
		t.Fatal(err)
	}
	if s.Life != 1736 || s.Stats["strength"] != 620 || s.Stats["agility"] != 40 {
		t.Fatalf("life/stats = %d %v", s.Life, s.Stats)
	}
	if _, ok := s.Stats["life"]; ok {
		t.Error("life doublé dans stats")
	}
	if s.Critical.ChancePct != 4 {
		t.Errorf("critique = %v, attendu 4 %%", s.Critical.ChancePct)
	}
	if len(s.Weapons) != 3 || len(s.Chips) != 12 {
		t.Fatalf("équipement = %d armes, %d puces", len(s.Weapons), len(s.Chips))
	}
	// laser : 43 + 16 jet, force 620 : ×7,2.
	laser := s.Weapons[0]
	if laser.Name != "laser" || laser.Cost != 6 {
		t.Fatalf("arme 0 = %+v", laser)
	}
	want := ScoutEffect{ID: 1, Name: "damage", Stat: "strength", Min: 310, Max: 425, Avg: 367.2}
	if laser.Effects[0] != want {
		t.Errorf("laser = %+v, attendu %+v", laser.Effects[0], want)
	}
	for _, c := range s.Chips {
		if c.Cooldown == nil {
			t.Errorf("puce %s sans recharge", c.Name)
		}
	}
}

func TestScoutEffectsByKind(t *testing.T) {
	stats := map[string]int{"magic": 300, "strength": 900}
	got := scoutEffects([]leekwars.Effect{
		{ID: 13, Value1: 10, Value2: 5, Turns: 3}, // poison : magie
		{ID: 31, Value1: 2},                       // raw_buff_mp : brut
		{ID: 99, Value1: 7, Value2: 1},            // inconnu : brut
	}, stats)
	want := []ScoutEffect{
		{ID: 13, Name: "poison", Stat: "magic", Min: 40, Max: 60, Avg: 50, Turns: 3},
		{ID: 31, Name: "raw_buff_mp", Min: 2, Max: 2, Avg: 2},
		{ID: 99, Name: "effect_99", Min: 7, Max: 8, Avg: 7.5},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("effet %d = %+v, attendu %+v", i, got[i], want[i])
		}
	}
}

func TestScoutVsUsFarmerFight(t *testing.T) {
	// Vu de l'éleveur 20225 : le combat d'éleveur 53996587 perdu contre 135146.
	v, err := ScoutVsUs(fixture(t, "leek_history_135146.json"), 135146, 20225, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if v.Victories != 0 || v.Draws != 0 || v.Defeats != 1 || len(v.Fights) != 1 {
		t.Fatalf("bilan = %+v", v)
	}
	f := v.Fights[0]
	if f.ID != 53996587 || f.Result != "defeat" || len(f.OurLeeks) != 4 {
		t.Errorf("combat = %+v", f)
	}
	if len(v.ByLeek) != 4 || v.ByLeek[0].LeekID != 23222 || v.ByLeek[0].Defeats != 1 {
		t.Errorf("par poireau = %+v", v.ByLeek)
	}
}

func TestScoutVsUsByLeek(t *testing.T) {
	// Un seul de nos poireaux connu, sans éleveur : seul ce poireau compte.
	v, err := ScoutVsUs(fixture(t, "leek_history_135146.json"), 135146, 0, []int{25075}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if v.Defeats != 1 || len(v.ByLeek) != 1 || v.ByLeek[0].LeekID != 25075 {
		t.Fatalf("bilan = %+v", v)
	}
}

func TestScoutVsUsNone(t *testing.T) {
	v, err := ScoutVsUs(fixture(t, "leek_history_135146.json"), 135146, 999, []int{999999}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if v.Victories+v.Draws+v.Defeats != 0 || len(v.Fights) != 0 {
		t.Fatalf("bilan = %+v", v)
	}
}
