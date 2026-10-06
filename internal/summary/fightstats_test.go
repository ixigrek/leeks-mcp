package summary

import (
	"slices"
	"strings"
	"testing"
)

// Combat 53994496 : plop2point0 perdu contre Imarmaleek (mage poison, bulbe soigneur).
func TestFightStatsSoloDefeat(t *testing.T) {
	s, err := FightStatsOf(fixture(t, "fight_53994496.json"), items(t), 135146, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Result != "defeat" || s.Duration != 12 || s.Name != "plop2point0" || s.Entity != 0 {
		t.Fatalf("en-tête : %+v", s)
	}
	me, them := s.Totals.Me, s.Totals.Them
	// Mort : 1772 PV de départ + 720 soignés.
	if me.DamageTaken != 2492 || them.DamageDealt != 2492 || me.Heal != 720 {
		t.Fatalf("dégâts reçus : %+v", me)
	}
	if me.PoisonReceived != 2447 || them.PoisonDealt != 2447 || them.DamageByItem["poison"] != 2447 {
		t.Fatalf("poison : moi %d, eux %d", me.PoisonReceived, them.PoisonDealt)
	}
	if me.DamageDealt != 2589 || me.DamageByItem["rhino"] != 629 || me.DamageByItem["iceberg"] != 1009 {
		t.Fatalf("dégâts infligés : %v", me.DamageByItem)
	}
	if me.ShieldsCast["armor"] == 0 || me.Shields != nil || them.ShieldsCast["helmet"] == 0 {
		t.Fatalf("boucliers en totaux : moi %v, eux %v", me.ShieldsCast, them.ShieldsCast)
	}
	t1, t4 := s.Turns[0], s.Turns[3]
	if t1.Distance != 19 || !t1.ZeroDamage || t1.Me.TPUnused != 16 || t1.Me.MPUsed != 3 {
		t.Fatalf("tour 1 : %+v", t1)
	}
	// Tour 4 : Adrénaline (+5 PT dans le tour), 20 PT utilisés sur 21.
	if t4.Me.TPUsed != 20 || t4.Me.TPUnused != 1 || t4.Me.WeaponShots != 3 || t4.ZeroDamage {
		t.Fatalf("tour 4 : %+v", t4.Me)
	}
	// Casque posé par le bulbe sur Imarmaleek.
	if sh := s.Turns[0].Them.Shields; len(sh) != 1 || sh[0].Item != "helmet" || sh[0].Kind != "absolute" || sh[0].Target != "Imarmaleek" {
		t.Fatalf("boucliers adverses tour 1 : %+v", sh)
	}
	if them.DamageTakenShielded != them.DamageTaken || me.DamageTakenShielded == 0 || me.DamageTakenShielded >= me.DamageTaken {
		t.Fatalf("dégâts sous bouclier : moi %d/%d, eux %d/%d", me.DamageTakenShielded, me.DamageTaken, them.DamageTakenShielded, them.DamageTaken)
	}
	if len(s.Flags) != 1 || s.Flags[0].Code != "tp_unused" || !slices.Equal(s.Flags[0].Turns, []int{3, 7, 8, 9}) {
		t.Fatalf("drapeaux : %+v", s.Flags)
	}
}

// Combat de boss 53996480 : Bottes de cuir lancées aux tours 7 et 12 sans tir d'arme.
func TestFightStatsBootsWithoutShot(t *testing.T) {
	s, err := FightStatsOf(fixture(t, "fight_53996480.json"), items(t), 135146, nil)
	if err != nil {
		t.Fatal(err)
	}
	var boots *StatsFlag
	for i := range s.Flags {
		if s.Flags[i].Code == "boots_without_shot" {
			boots = &s.Flags[i]
		}
	}
	if boots == nil || !slices.Equal(boots.Turns, []int{7, 12}) {
		t.Fatalf("drapeaux : %+v", s.Flags)
	}
	if s.Result != "win" || len(s.Notes) == 0 || !strings.Contains(s.Notes[0], "téléportations") {
		t.Fatalf("résultat %s, notes %v", s.Result, s.Notes)
	}
}

func TestFightStatsUnknownLeek(t *testing.T) {
	if _, err := FightStatsOf(fixture(t, "fight_53994496.json"), items(t), 1, nil); err == nil {
		t.Fatal("erreur attendue pour un poireau absent")
	}
}

// Drapeaux tirés de l'équipement et des séries de tours sans dégât.
func TestStatsFlagsShieldsAndGuard(t *testing.T) {
	s := &FightStats{}
	for i := 1; i <= 8; i++ {
		s.Turns = append(s.Turns, StatsTurn{Turn: i, ZeroDamage: i >= 2 && i <= 6})
	}
	s.Totals.Me.DamageTaken = 300
	// shield (id 20) et laser (item 42).
	equip := &Equipment{Chips: []int{20}, Weapons: []int{42}}
	flags := statsFlags(s, items(t), equip, flagInput{})
	codes := map[string]StatsFlag{}
	for _, f := range flags {
		codes[f.Code] = f
	}
	if g, ok := codes["long_guard"]; !ok || !slices.Equal(g.Turns, []int{2, 3, 4, 5, 6}) || !strings.Contains(g.Detail, "tours 2-6") {
		t.Fatalf("long_guard : %+v", flags)
	}
	if _, ok := codes["shields_never_cast"]; !ok {
		t.Fatalf("shields_never_cast attendu : %+v", flags)
	}
	flags = statsFlags(s, items(t), equip, flagInput{shieldCast: true})
	for _, f := range flags {
		if f.Code == "shields_never_cast" {
			t.Fatal("bouclier lancé : pas de drapeau")
		}
	}
}

func TestCellDistance(t *testing.T) {
	// Pas d'une case : ±17 ou ±18 sur une carte de largeur 18.
	for _, c := range [][3]int{{55, 73, 1}, {73, 56, 1}, {37, 37, 0}, {37, 258, 22}} {
		if d := cellDistance(c[0], c[1], 18); d != c[2] {
			t.Fatalf("distance(%d, %d) = %d, attendu %d", c[0], c[1], d, c[2])
		}
	}
}

func TestLeekEquipment(t *testing.T) {
	e, err := LeekEquipment(fixture(t, "leek_135146.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Weapons) == 0 || len(e.Chips) == 0 || !slices.Contains(e.Chips, 14) {
		t.Fatalf("équipement : %+v", e)
	}
}
