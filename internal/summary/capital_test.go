package summary

import "testing"

func TestBonusToCapitalFollowsSteps(t *testing.T) {
	cases := []struct {
		stat           string
		bonus, capital int
	}{
		{"strength", 0, 0},
		{"strength", 20, 10},   // 2 points par capital sous 200
		{"strength", 600, 700}, // 100 + 200 + 400
		{"life", 840, 210},     // 4 points par capital sous 1000
		{"tp", 6, 255},         // 30+35+40+45+50+55
		{"mp", 2, 60},          // 20+40
		{"ram", 2, 50},         // 20+30
		{"frequency", 7, 7},
		{"strength", -3, 0},
	}
	for _, c := range cases {
		if got := BonusToCapital(c.stat, c.bonus); got != c.capital {
			t.Errorf("%s/%d = %d, attendu %d", c.stat, c.bonus, got, c.capital)
		}
	}
	if TotalCapital(248) != 1375 || TotalCapital(1) != 50 || TotalCapital(301) != 50+300*5+135+95 {
		t.Fatalf("capital total : %d %d %d", TotalCapital(248), TotalCapital(1), TotalCapital(301))
	}
	if !IsStat("life") || IsStat("speed") {
		t.Fatal("IsStat")
	}
}

func TestBuildAndCapitalSpent(t *testing.T) {
	b, err := Build(fixture(t, "leek_private_135146.json"))
	if err != nil {
		t.Fatal(err)
	}
	if b.ID != 135146 || b.Level != 233 || b.Capital != 25 || b.MaxWeapons != 4 {
		t.Fatalf("build : %+v", b)
	}
	if len(b.Weapons) != 3 || b.Weapons[0] != 42 || len(b.Chips) != 12 || b.Chips[0] != 14 {
		t.Fatalf("équipement : %v %v", b.Weapons, b.Chips)
	}
	if len(b.Components) != 6 || b.Components[5].Index != 5 || b.Components[5].Template != 313 || string(b.Components[0].Stats) != "null" {
		t.Fatalf("composants : %+v", b.Components)
	}
	spent := CapitalSpent(b.Level, b.Stats)
	// life 1636 - 796 = 840 → 210 ; strength 600 → 700 ; tp 16-10 = 6 → 255 ; mp 5-3 = 2 → 60 ; ram 8-6 = 2 → 50.
	want := map[string]int{"life": 210, "strength": 700, "tp": 255, "mp": 60, "ram": 50}
	if len(spent) != len(want) {
		t.Fatalf("capital dépensé : %v", spent)
	}
	for k, v := range want {
		if spent[k] != v {
			t.Fatalf("%s : %d, attendu %d (%v)", k, spent[k], v, spent)
		}
	}
	total := 0
	for _, v := range spent {
		total += v
	}
	if total+b.Capital != TotalCapital(b.Level) {
		t.Fatalf("dépensé %d + restant %d ≠ total %d", total, b.Capital, TotalCapital(b.Level))
	}
}
