package summary

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// AIFile est une IA en ligne de l'arborescence de l'éleveur.
type AIFile struct {
	Path  string `json:"path"`
	Valid bool   `json:"valid"`
	Lines int    `json:"lines"`
	Mtime int64  `json:"mtime"`
}

// AITreeSummary résume ai/get-farmer-tree : les IA en ligne et l'IA jouée par
// chaque poireau (id → nom). La corbeille (bin) et les dossiers vides sont omis.
type AITreeSummary struct {
	LeekAIs map[string]string `json:"leek_ais"`
	Files   []AIFile          `json:"files"`
}

// AITree lit ai/get-farmer-tree.
func AITree(body []byte) (*AITreeSummary, error) {
	var raw struct {
		Files []struct {
			Path       string `json:"path"`
			Valid      bool   `json:"valid"`
			TotalLines int    `json:"total_lines"`
			Mtime      int64  `json:"mtime"`
		} `json:"files"`
		LeekAIs map[string]string `json:"leek_ais"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("ai/get-farmer-tree : %w", err)
	}
	s := &AITreeSummary{LeekAIs: raw.LeekAIs, Files: make([]AIFile, 0, len(raw.Files))}
	if s.LeekAIs == nil {
		s.LeekAIs = map[string]string{}
	}
	for _, f := range raw.Files {
		s.Files = append(s.Files, AIFile{Path: f.Path, Valid: f.Valid, Lines: f.TotalLines, Mtime: f.Mtime})
	}
	sort.Slice(s.Files, func(i, j int) bool { return s.Files[i].Path < s.Files[j].Path })
	return s, nil
}

// Has dit si une IA de ce nom existe en ligne.
func (t *AITreeSummary) Has(path string) bool {
	for _, f := range t.Files {
		if f.Path == path {
			return true
		}
	}
	return false
}

// PlayedBy renvoie les ids des poireaux qui jouent cette IA, triés.
func (t *AITreeSummary) PlayedBy(path string) []string {
	var ids []string
	for id, ai := range t.LeekAIs {
		if ai == path {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

var versionRe = regexp.MustCompile(`(?m)^\s*global\s+\w*VERSION\s*=\s*"([^"]*)"`)

// AIVersion extrait la première constante VERSION du script (VERSION, g_VERSION
// des _grp, n_VERSION des _nasu), "" si absente.
func AIVersion(code string) string {
	if m := versionRe.FindStringSubmatch(code); m != nil {
		return m[1]
	}
	return ""
}

// AIRead lit ai/read.
func AIRead(body []byte) (code string, mtime int64, err error) {
	var raw struct {
		Code  *string `json:"code"`
		Mtime int64   `json:"mtime"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", 0, fmt.Errorf("ai/read : %w", err)
	}
	if raw.Code == nil {
		return "", 0, fmt.Errorf("ai/read : réponse sans code")
	}
	return *raw.Code, raw.Mtime, nil
}

// AIProblem est une erreur de compilation renvoyée par ai/write.
type AIProblem struct {
	File   string   `json:"file,omitempty"` // seulement si ce n'est pas le fichier poussé
	Line   int      `json:"line"`
	Col    int      `json:"col"`
	Code   int      `json:"code"`
	Params []string `json:"params,omitempty"`
}

// AIWriteResult résume ai/write : les problems du fichier poussé et la validité
// des IA qui l'incluent.
type AIWriteResult struct {
	Problems  []AIProblem     `json:"problems"`
	Includers map[string]bool `json:"includers,omitempty"`
}

// AIWrite lit la réponse de ai/write pour le fichier path. Un problem est une
// liste [niveau, fichier, ligne, colonne, ligne de fin, colonne de fin, code, params].
func AIWrite(body []byte, path string) (*AIWriteResult, error) {
	var raw struct {
		Result map[string]struct {
			Problems []json.RawMessage `json:"problems"`
			Valid    *bool             `json:"valid"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("ai/write : %w", err)
	}
	own, found := raw.Result[path]
	if !found {
		return nil, fmt.Errorf("ai/write : réponse sans entrée pour %q", path)
	}
	res := &AIWriteResult{Problems: []AIProblem{}}
	for _, p := range own.Problems {
		res.Problems = append(res.Problems, aiProblem(p, path))
	}
	for name, r := range raw.Result {
		if name != path && r.Valid != nil {
			if res.Includers == nil {
				res.Includers = map[string]bool{}
			}
			res.Includers[name] = *r.Valid
		}
	}
	return res, nil
}

func aiProblem(data json.RawMessage, path string) AIProblem {
	var fields []json.RawMessage
	if json.Unmarshal(data, &fields) != nil || len(fields) < 7 {
		// Forme inconnue : la garder lisible plutôt que de la perdre.
		return AIProblem{Params: []string{strings.TrimSpace(string(data))}}
	}
	var p AIProblem
	json.Unmarshal(fields[1], &p.File)
	json.Unmarshal(fields[2], &p.Line)
	json.Unmarshal(fields[3], &p.Col)
	json.Unmarshal(fields[6], &p.Code)
	if len(fields) > 7 {
		var params []any
		json.Unmarshal(fields[7], &params)
		for _, v := range params {
			p.Params = append(p.Params, fmt.Sprint(v))
		}
	}
	if p.File == path {
		p.File = ""
	}
	return p
}

// FirstDiffLine renvoie le numéro (à partir de 1) de la première ligne qui
// diffère entre a et b, 0 s'ils sont identiques.
func FirstDiffLine(a, b string) int {
	if a == b {
		return 0
	}
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(la) && i < len(lb); i++ {
		if la[i] != lb[i] {
			return i + 1
		}
	}
	if len(la) < len(lb) {
		return len(la) + 1
	}
	return len(lb) + 1
}
