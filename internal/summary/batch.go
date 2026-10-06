package summary

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/ixigrek/leeks-mcp/internal/leekwars"
)

// Résultats d'un combat vu par un poireau.
const (
	ResultWin    = "win"
	ResultDraw   = "draw"
	ResultDefeat = "defeat"
)

// Outcome est le bilan d'un poireau dans un combat terminé.
type Outcome struct {
	Fight    int
	Result   string
	Turns    int
	Life     int      // PV en fin de combat
	Versions []string // VERSION affichées au tour 1, dans l'ordre, sans doublon
}

// FightOutcome lit le résultat, la durée et les PV restants du poireau leekID
// dans un rapport terminé, et les VERSION qu'il a affichées au tour 1 dans les
// logs (logsJSON nil = logs indisponibles, Versions vide).
func FightOutcome(reportJSON, logsJSON []byte, items *leekwars.Items, leekID int) (*Outcome, error) {
	raw, err := parseFightReport(reportJSON)
	if err != nil {
		return nil, err
	}
	team := 0
	for _, l := range raw.Leeks1 {
		if l.ID == leekID {
			team = 1
		}
	}
	for _, l := range raw.Leeks2 {
		if l.ID == leekID {
			team = 2
		}
	}
	if team == 0 {
		return nil, fmt.Errorf("le poireau %d ne participe pas au combat %d", leekID, raw.ID)
	}
	s, err := Fight(reportJSON, items, 0)
	if err != nil {
		return nil, err
	}
	o := &Outcome{Fight: raw.ID, Turns: s.Duration, Versions: []string{}}
	switch raw.Winner {
	case 0:
		o.Result = ResultDraw
	case team:
		o.Result = ResultWin
	default:
		o.Result = ResultDefeat
	}
	for _, e := range s.Entities {
		if e.LeekID == leekID && !e.Summon {
			o.Life = e.LifeEnd
		}
	}
	if logsJSON != nil {
		logs, err := Logs(logsJSON, reportJSON, leekID)
		if err != nil {
			return nil, err
		}
		o.Versions = turnOneVersions(logs)
	}
	return o, nil
}

// versionTag repère « vX… » dans une ligne de log (v2026-10-06d, vG00-…, vN-…) ;
// X chiffre ou majuscule, pour ne pas prendre « vers » ou « valeur ».
var versionTag = regexp.MustCompile(`(?:^|\s)v([0-9A-Z][\w.-]*)`)

func turnOneVersions(logs *LogsSummary) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, t := range logs.Turns {
		if t.Turn != 1 {
			continue
		}
		for _, e := range t.Entries {
			for _, m := range versionTag.FindAllStringSubmatch(e.Text, -1) {
				if !seen[m[1]] {
					seen[m[1]] = true
					out = append(out, m[1])
				}
			}
		}
	}
	return out
}

// BatchSummary agrège les bilans d'un lot de combats.
type BatchSummary struct {
	Fights      int            `json:"fights"`
	Wins        int            `json:"wins"`
	Draws       int            `json:"draws"`
	Defeats     int            `json:"defeats"`
	AvgTurns    float64        `json:"avg_turns"`
	AvgLife     float64        `json:"avg_life"`
	AvgLifeWins float64        `json:"avg_life_wins"`
	Versions    map[string]int `json:"versions"`
	DefeatIDs   []int          `json:"defeat_ids"`
}

// unknownVersion compte les combats sans VERSION lisible au tour 1.
const unknownVersion = "?"

// Batch agrège des bilans : moyennes arrondies au dixième, VERSION jouées
// (plusieurs au tour 1, ex. _grp + _nasu + IA, jointes par « + ») par nombre de combats.
func Batch(outcomes []Outcome) *BatchSummary {
	b := &BatchSummary{Fights: len(outcomes), Versions: map[string]int{}, DefeatIDs: []int{}}
	var turns, life, lifeWins int
	for _, o := range outcomes {
		switch o.Result {
		case ResultWin:
			b.Wins++
			lifeWins += o.Life
		case ResultDraw:
			b.Draws++
		default:
			b.Defeats++
			b.DefeatIDs = append(b.DefeatIDs, o.Fight)
		}
		turns += o.Turns
		life += o.Life
		v := strings.Join(o.Versions, " + ")
		if v == "" {
			v = unknownVersion
		}
		b.Versions[v]++
	}
	b.AvgTurns = mean(turns, b.Fights)
	b.AvgLife = mean(life, b.Fights)
	b.AvgLifeWins = mean(lifeWins, b.Wins)
	return b
}

func mean(sum, n int) float64 {
	if n == 0 {
		return 0
	}
	return math.Round(float64(sum)/float64(n)*10) / 10
}
