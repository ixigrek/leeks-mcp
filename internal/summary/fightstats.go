package summary

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
)

// Types d'effet (champ type des actions ADD_*_EFFECT et effects[].id des objets).
const (
	effRelativeShield = 5
	effAbsoluteShield = 6
	effBuffMP         = 7
	effTeleport       = 10
	effPermutation    = 11
	effPoison         = 13
	effRawBuffMP      = 31
)

// Seuil du drapeau long_guard : tours d'affilée sans aucune perte de PV.
const longGuardTurns = 5

// ShieldCast est un bouclier posé.
type ShieldCast struct {
	Item   string `json:"item"`
	Kind   string `json:"kind"` // absolute ou relative
	Value  int    `json:"value"`
	Turns  int    `json:"turns"`
	Target string `json:"target,omitempty"` // absent : le lanceur lui-même
}

// SideStats chiffre l'activité d'un camp sur un tour ou sur tout le combat.
// Les champs nuls sont omis.
type SideStats struct {
	TPUsed              int            `json:"tp_used,omitempty"`
	TPUnused            int            `json:"tp_unused,omitempty"`
	TPLost              int            `json:"tp_lost,omitempty"`
	MPUsed              int            `json:"mp_used,omitempty"`
	MPUnused            int            `json:"mp_unused,omitempty"`
	MPLost              int            `json:"mp_lost,omitempty"`
	WeaponShots         int            `json:"weapon_shots,omitempty"`
	DamageDealt         int            `json:"damage_dealt,omitempty"`
	DamageByItem        map[string]int `json:"damage_by_item,omitempty"`
	DamageTaken         int            `json:"damage_taken,omitempty"`
	DamageTakenShielded int            `json:"damage_taken_shielded,omitempty"`
	Shields             []ShieldCast   `json:"shields,omitempty"`
	Heal                int            `json:"heal,omitempty"`
	PoisonReceived      int            `json:"poison_received,omitempty"`
	PoisonDealt         int            `json:"poison_dealt,omitempty"`
}

func (s *SideStats) add(o SideStats) {
	s.TPUsed += o.TPUsed
	s.TPUnused += o.TPUnused
	s.TPLost += o.TPLost
	s.MPUsed += o.MPUsed
	s.MPUnused += o.MPUnused
	s.MPLost += o.MPLost
	s.WeaponShots += o.WeaponShots
	s.DamageDealt += o.DamageDealt
	for k, v := range o.DamageByItem {
		if s.DamageByItem == nil {
			s.DamageByItem = map[string]int{}
		}
		s.DamageByItem[k] += v
	}
	s.DamageTaken += o.DamageTaken
	s.DamageTakenShielded += o.DamageTakenShielded
	s.Shields = append(s.Shields, o.Shields...)
	s.Heal += o.Heal
	s.PoisonReceived += o.PoisonReceived
	s.PoisonDealt += o.PoisonDealt
}

func (s *SideStats) dealt(item string, pv int) {
	s.DamageDealt += pv
	if s.DamageByItem == nil {
		s.DamageByItem = map[string]int{}
	}
	s.DamageByItem[item] += pv
}

// StatsTurn est la vue chiffrée d'un tour : le camp du poireau (me) et le camp adverse.
type StatsTurn struct {
	Turn int `json:"turn"`
	// Distance en cases entre le poireau et l'ennemi vivant le plus proche, à la fin
	// du tour du poireau (absente s'il n'a pas joué).
	Distance   int       `json:"distance,omitempty"`
	ZeroDamage bool      `json:"zero_damage,omitempty"`
	Me         SideStats `json:"me"`
	Them       SideStats `json:"them"`
}

// StatsFlag est un motif de défaite repéré dans le combat.
type StatsFlag struct {
	Code   string `json:"code"`
	Turns  []int  `json:"turns,omitempty"`
	Detail string `json:"detail"`
}

// FightStats est le diagnostic chiffré d'un combat du point de vue d'un poireau.
type FightStats struct {
	ID       int    `json:"id"`
	LeekID   int    `json:"leek_id"`
	Entity   int    `json:"entity"`
	Name     string `json:"name"`
	Result   string `json:"result"`
	Duration int    `json:"duration"`
	Totals   struct {
		Me   SideStats `json:"me"`
		Them SideStats `json:"them"`
	} `json:"totals"`
	Flags []StatsFlag `json:"flags"`
	Turns []StatsTurn `json:"turns"`
	Notes []string    `json:"notes,omitempty"`
}

// Equipment est l'équipement connu du poireau : ids de puces et items d'armes,
// ceux de leek/get. Nil si inconnu.
type Equipment struct {
	Chips   []int
	Weapons []int
}

// statsEntity suit une entité pendant le combat.
type statsEntity struct {
	team, cell, tp, mp int
	summon, dead       bool
}

// FightStatsOf calcule le diagnostic d'un rapport terminé pour le poireau leekID.
// equip (facultatif) sert aux drapeaux tp_unused et shields_never_cast.
func FightStatsOf(body []byte, items *leekwars.Items, leekID int, equip *Equipment) (*FightStats, error) {
	raw, err := parseFightReport(body)
	if err != nil {
		return nil, err
	}
	var mapInfo struct {
		Data struct {
			Leeks []struct {
				ID      int `json:"id"`
				CellPos int `json:"cellPos"`
			} `json:"leeks"`
			Map struct {
				Width int `json:"width"`
			} `json:"map"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &mapInfo); err != nil {
		return nil, fmt.Errorf("fight/get : %w", err)
	}
	width := mapInfo.Data.Map.Width
	if width == 0 {
		width = 18
	}
	names := raw.leekIDByName()
	ents := map[int]*statsEntity{}
	me := -1
	s := &FightStats{ID: raw.ID, LeekID: leekID, Flags: []StatsFlag{}, Turns: []StatsTurn{}}
	for _, l := range raw.Data.Leeks {
		ents[l.ID] = &statsEntity{team: l.Team, tp: l.TP, mp: l.MP, summon: l.Summon}
		if !l.Summon && names[l.Name] == leekID {
			me, s.Entity, s.Name = l.ID, l.ID, l.Name
		}
	}
	if me < 0 {
		return nil, fmt.Errorf("le poireau %d ne participe pas au combat %d", leekID, raw.ID)
	}
	for _, l := range mapInfo.Data.Leeks {
		if e := ents[l.ID]; e != nil {
			e.cell = l.CellPos
		}
	}
	myTeam := ents[me].team
	switch {
	case raw.Winner == 0:
		s.Result = "draw"
	case raw.Winner == myTeam:
		s.Result = "win"
	default:
		s.Result = "defeat"
	}
	side := func(t *StatsTurn, id int) *SideStats {
		if e := ents[id]; e != nil && e.team == myTeam {
			return &t.Me
		}
		return &t.Them
	}
	other := func(t *StatsTurn, id int) *SideStats {
		if e := ents[id]; e != nil && e.team == myTeam {
			return &t.Them
		}
		return &t.Me
	}

	// Effets actifs : id de log → cible, pour les boucliers.
	shieldOn := map[int]int{}
	shieldCount := map[int]int{}
	lastItem := map[int]string{}
	weapon := map[int]int{}
	myBoots, myShots, myUnused := map[int]bool{}, map[int]bool{}, map[int]int{}
	// Distance et PM du poireau au début de son tour, pour savoir si l'ennemi était atteignable.
	myReach := map[int][2]int{}
	usedRange := 0
	myShieldCast := false
	usedCosts := map[int]bool{}
	teleports := 0

	turn := &StatsTurn{Turn: 1}
	lifeLost := false
	caster := -1
	var tpUsed, mpUsed int
	// endEntityTurn clôt le tour de l'entité active. finished : la fin de tour a été
	// lue, e.tp/e.mp portent alors les PT et PM disponibles ce tour-ci (buffs du tour
	// compris, ex. Adrénaline) ; sinon (mort en cours de tour) rien n'est compté comme inutilisé.
	endEntityTurn := func(finished bool) {
		if caster < 0 {
			return
		}
		e := ents[caster]
		st := side(turn, caster)
		st.TPUsed += tpUsed
		st.MPUsed += mpUsed
		if finished {
			st.TPUnused += max(e.tp-tpUsed, 0)
			st.MPUnused += max(e.mp-mpUsed, 0)
		}
		if caster == me {
			turn.Distance = nearestEnemy(ents, me, width)
			if finished {
				myUnused[turn.Turn] = max(e.tp-tpUsed, 0)
			}
		}
		caster, tpUsed, mpUsed = -1, 0, 0
	}
	closeTurn := func() {
		endEntityTurn(false)
		turn.ZeroDamage = !lifeLost
		s.Turns = append(s.Turns, *turn)
		s.Totals.Me.add(turn.Me)
		s.Totals.Them.add(turn.Them)
	}
	damage := func(victim, pv int, item string, credit int) {
		if e := ents[victim]; e == nil {
			return
		}
		lifeLost = lifeLost || pv > 0
		taken := side(turn, victim)
		taken.DamageTaken += pv
		if shieldCount[victim] > 0 {
			taken.DamageTakenShielded += pv
		}
		// Crédit au camp adverse de la victime (les dégâts sur son propre camp ne comptent pas).
		if credit >= 0 && ents[credit] != nil && ents[credit].team == ents[victim].team {
			return
		}
		other(turn, victim).dealt(item, pv)
	}

	for _, a := range raw.Data.Actions {
		if len(a) == 0 {
			continue
		}
		switch argInt(a, 0) {
		case actNewTurn:
			closeTurn()
			turn, lifeLost = &StatsTurn{Turn: argInt(a, 1)}, false
		case actLeekTurn:
			endEntityTurn(false)
			caster = argInt(a, 1)
			if caster == me {
				myReach[turn.Turn] = [2]int{nearestEnemy(ents, me, width), ents[me].mp}
			}
		case actEndTurn:
			id := argInt(a, 1)
			if e := ents[id]; e != nil {
				e.tp, e.mp = argInt(a, 2), argInt(a, 3)
			}
			if id == caster {
				endEntityTurn(true)
			}
		case actSummon:
			if e := ents[argInt(a, 2)]; e != nil {
				e.cell = argInt(a, 3)
			}
		case actMoveTo:
			id := argInt(a, 1)
			if e := ents[id]; e != nil {
				e.cell = argInt(a, 2)
			}
			if id == caster {
				mpUsed += argLen(a, 3)
			}
		case actSetWeapon:
			if caster >= 0 {
				weapon[caster] = argInt(a, 1)
				tpUsed++
			}
		case actUseWeapon:
			if caster < 0 {
				continue
			}
			w := items.WeaponByID(weapon[caster])
			name := items.WeaponName(weapon[caster])
			if w != nil {
				tpUsed += w.Cost
				if caster == me {
					usedCosts[w.Cost] = true
					usedRange = max(usedRange, w.MaxRange)
				}
			}
			lastItem[caster] = name
			side(turn, caster).WeaponShots++
			if caster == me {
				myShots[turn.Turn] = true
			}
		case actUseChip:
			if caster < 0 {
				continue
			}
			c := items.ChipByTemplate(argInt(a, 1))
			lastItem[caster] = items.ChipNameByTemplate(argInt(a, 1))
			if c == nil {
				continue
			}
			tpUsed += c.Cost
			if caster == me && hasDamage(c.Effects) {
				usedCosts[c.Cost] = true
				usedRange = max(usedRange, c.MaxRange)
			}
			cell := argInt(a, 2)
			for _, eff := range c.Effects {
				switch eff.ID {
				case effBuffMP, effRawBuffMP:
					if caster == me {
						myBoots[turn.Turn] = true
					}
				case effTeleport:
					ents[caster].cell = cell
					teleports++
				case effPermutation:
					for _, o := range ents {
						if !o.dead && o.cell == cell && o != ents[caster] {
							o.cell, ents[caster].cell = ents[caster].cell, cell
							break
						}
					}
					teleports++
				}
			}
		case actTPLost:
			side(turn, argInt(a, 1)).TPLost += argInt(a, 2)
		case actMPLost:
			side(turn, argInt(a, 1)).MPLost += argInt(a, 2)
		case actLifeLost, actLifeDamage, actNovaDamage:
			item := lastItem[caster]
			if caster < 0 || item == "" {
				item = "autre"
			}
			damage(argInt(a, 1), argInt(a, 2), item, caster)
		case actDamageReturn:
			damage(argInt(a, 1), argInt(a, 2), "renvoi", -1)
		case actAftereffect:
			damage(argInt(a, 1), argInt(a, 2), "aftereffect", -1)
		case actPoisonDamage:
			victim, pv := argInt(a, 1), argInt(a, 2)
			damage(victim, pv, "poison", -1)
			if ents[victim] != nil {
				side(turn, victim).PoisonReceived += pv
				other(turn, victim).PoisonDealt += pv
			}
		case actCare:
			if ents[argInt(a, 1)] != nil {
				side(turn, argInt(a, 1)).Heal += argInt(a, 2)
			}
		case actAddWeaponEff, actAddChipEff:
			typ := argInt(a, 5)
			if typ != effRelativeShield && typ != effAbsoluteShield {
				continue
			}
			by, target := argInt(a, 3), argInt(a, 4)
			if ents[by] == nil || ents[target] == nil {
				continue
			}
			item := items.ChipName(argInt(a, 1))
			if argInt(a, 0) == actAddWeaponEff {
				item = items.WeaponName(argInt(a, 1))
			}
			sc := ShieldCast{Item: item, Kind: "absolute", Value: argInt(a, 6), Turns: argInt(a, 7)}
			if typ == effRelativeShield {
				sc.Kind = "relative"
			}
			if target != by {
				sc.Target = entityName(raw, target)
			}
			st := side(turn, by)
			st.Shields = append(st.Shields, sc)
			shieldOn[argInt(a, 2)] = target
			shieldCount[target]++
			if by == me {
				myShieldCast = true
			}
		case actRemoveEffect:
			if target, ok := shieldOn[argInt(a, 1)]; ok {
				delete(shieldOn, argInt(a, 1))
				shieldCount[target]--
			}
		case actPlayerDead:
			if e := ents[argInt(a, 1)]; e != nil {
				e.dead = true
			}
		}
	}
	closeTurn()
	s.Duration = len(s.Turns)
	if teleports > 0 {
		s.Notes = append(s.Notes, fmt.Sprintf("%d téléportations ou permutations : distances approchées", teleports))
	}
	s.Flags = statsFlags(s, items, equip, flagInput{usedCosts, usedRange, myUnused, myReach, myBoots, myShots, myShieldCast})
	if equip != nil {
		s.Notes = append(s.Notes, "équipement : celui du poireau aujourd'hui, pas forcément celui du combat")
	}
	return s, nil
}

// flagInput porte ce que le parcours du rapport a relevé sur le poireau.
type flagInput struct {
	usedCosts  map[int]bool
	usedRange  int
	unused     map[int]int
	reach      map[int][2]int // tour → distance et PM au début de son tour
	boots      map[int]bool
	shots      map[int]bool
	shieldCast bool
}

func statsFlags(s *FightStats, items *leekwars.Items, equip *Equipment, in flagInput) []StatsFlag {
	flags := []StatsFlag{}
	// PT inutilisés : au moins le coût de l'objet d'attaque le moins cher de
	// l'équipement (à défaut, des objets d'attaque utilisés dans le combat), sur un
	// tour où l'ennemi était à portée de PM + portée max d'un objet d'attaque.
	cheapest, maxRange := 0, 0
	consider := func(cost, rng int) {
		if cost > 0 && (cheapest == 0 || cost < cheapest) {
			cheapest = cost
		}
		maxRange = max(maxRange, rng)
	}
	hasShield := false
	if equip != nil {
		for _, id := range equip.Chips {
			if c := items.ChipByID(id); c != nil {
				if hasDamage(c.Effects) {
					consider(c.Cost, c.MaxRange)
				}
				for _, e := range c.Effects {
					if e.ID == effRelativeShield || e.ID == effAbsoluteShield {
						hasShield = true
					}
				}
			}
		}
		for _, item := range equip.Weapons {
			if w := items.WeaponByItem(item); w != nil {
				consider(w.Cost, w.MaxRange)
			}
		}
	}
	if cheapest == 0 {
		for c := range in.usedCosts {
			consider(c, in.usedRange)
		}
	}
	if cheapest > 0 && maxRange > 0 {
		var turns []int
		total := 0
		for _, t := range s.Turns {
			left, played := in.unused[t.Turn]
			r := in.reach[t.Turn]
			if played && left >= cheapest && r[0] > 0 && r[0] <= r[1]+maxRange {
				turns = append(turns, t.Turn)
				total += left
			}
		}
		if len(turns) > 0 {
			flags = append(flags, StatsFlag{Code: "tp_unused", Turns: turns,
				Detail: fmt.Sprintf("%d tours avec l'ennemi atteignable (PM + portée %d) et au moins %d PT inutilisés (objet d'attaque le moins cher), %d PT au total", len(turns), maxRange, cheapest, total)})
		}
	}
	// Garde qui s'éternise : séries de tours sans aucune perte de PV.
	var runs []string
	var guardTurns []int
	for i := 0; i < len(s.Turns); {
		j := i
		for j < len(s.Turns) && s.Turns[j].ZeroDamage {
			j++
		}
		if j-i >= longGuardTurns {
			runs = append(runs, fmt.Sprintf("tours %d-%d", s.Turns[i].Turn, s.Turns[j-1].Turn))
			for k := i; k < j; k++ {
				guardTurns = append(guardTurns, s.Turns[k].Turn)
			}
		}
		i = max(j, i+1)
	}
	if len(runs) > 0 {
		flags = append(flags, StatsFlag{Code: "long_guard", Turns: guardTurns,
			Detail: fmt.Sprintf("au moins %d tours d'affilée à 0 dégât des deux côtés : %s", longGuardTurns, strings.Join(runs, ", "))})
	}
	if hasShield && !in.shieldCast && s.Totals.Me.DamageTaken > 0 {
		flags = append(flags, StatsFlag{Code: "shields_never_cast",
			Detail: fmt.Sprintf("puces de bouclier équipées, jamais lancées, %d dégâts reçus", s.Totals.Me.DamageTaken)})
	}
	var bootTurns []int
	for t := range in.boots {
		if !in.shots[t] {
			bootTurns = append(bootTurns, t)
		}
	}
	if len(bootTurns) > 0 {
		sort.Ints(bootTurns)
		flags = append(flags, StatsFlag{Code: "boots_without_shot", Turns: bootTurns,
			Detail: fmt.Sprintf("%d tours avec une puce de PM lancée sans tir d'arme", len(bootTurns))})
	}
	return flags
}

// nearestEnemy renvoie la distance en cases entre l'entité id et l'ennemi vivant
// le plus proche (poireaux d'abord, invocations à défaut), 0 s'il n'y en a pas.
func nearestEnemy(ents map[int]*statsEntity, id, width int) int {
	me := ents[id]
	best, bestSummon := 0, 0
	for _, e := range ents {
		if e.team == me.team || e.dead {
			continue
		}
		d := cellDistance(me.cell, e.cell, width)
		if e.summon {
			if bestSummon == 0 || d < bestSummon {
				bestSummon = d
			}
		} else if best == 0 || d < best {
			best = d
		}
	}
	if best == 0 {
		return bestSummon
	}
	return best
}

// cellDistance est la distance en cases (déplacements) entre deux cellules d'une
// carte LeekWars de largeur width : les lignes alternent width et width-1 cellules,
// un pas vaut ±(width-1) ou ±width.
func cellDistance(a, b, width int) int {
	ax, ay := cellXY(a, width)
	bx, by := cellXY(b, width)
	return max(abs(ax-bx), abs(ay-by))
}

func cellXY(cell, width int) (int, int) {
	w2 := 2*width - 1
	row, col := cell/w2, cell%w2
	if col < width {
		return 2 * col, 2 * row
	}
	return 2*(col-width) + 1, 2*row + 1
}

// hasDamage dit si un objet inflige des dégâts directs ou du poison.
func hasDamage(effects []leekwars.Effect) bool {
	for _, e := range effects {
		if e.ID == 1 || e.ID == effPoison {
			return true
		}
	}
	return false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func entityName(raw *rawFightReport, id int) string {
	for _, l := range raw.Data.Leeks {
		if l.ID == id {
			return l.Name
		}
	}
	return fmt.Sprintf("entité %d", id)
}

// LeekEquipment lit l'équipement actuel d'un poireau dans leek/get : ids de
// puces et items d'armes (champ template des deux listes).
func LeekEquipment(leekJSON []byte) (*Equipment, error) {
	var raw struct {
		Weapons []struct {
			Template int `json:"template"`
		} `json:"weapons"`
		Chips []struct {
			Template int `json:"template"`
		} `json:"chips"`
	}
	if err := json.Unmarshal(leekJSON, &raw); err != nil {
		return nil, fmt.Errorf("leek/get : %w", err)
	}
	e := &Equipment{}
	for _, w := range raw.Weapons {
		e.Weapons = append(e.Weapons, w.Template)
	}
	for _, c := range raw.Chips {
		e.Chips = append(e.Chips, c.Template)
	}
	return e, nil
}
