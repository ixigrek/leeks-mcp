package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ixigrek/leeks-mcp/internal/summary"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Les IA en ligne portent le nom de leur fichier local sans l'extension .leek.
// Les bibliothèques _grp puis _nasu se poussent avant l'IA qui les inclut.

const leekExt = ".leek"

func (a *app) aiTree(ctx context.Context, _ *mcp.CallToolRequest, in rawFlag) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	body, err := a.client.Get(ctx, "ai/get-farmer-tree")
	if err != nil {
		return fail(err)
	}
	if in.Raw {
		return raw(body)
	}
	s, err := summary.AITree(body)
	if err != nil {
		return fail(err)
	}
	return ok(s)
}

type aiReadArgs struct {
	Name string `json:"name" jsonschema:"nom de l'IA en ligne, sans .leek (ex. force_grp)"`
	File string `json:"file,omitempty" jsonschema:"fichier local à comparer : renvoie seulement identique ou non et les VERSION, sans le code"`
	rawFlag
}

type aiReadResult struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Mtime   int64  `json:"mtime"`
	Lines   int    `json:"lines"`
	Code    string `json:"code"`
}

type aiCompareResult struct {
	Name          string `json:"name"`
	File          string `json:"file"`
	Identical     bool   `json:"identical"`
	OnlineVersion string `json:"online_version"`
	LocalVersion  string `json:"local_version"`
	FirstDiffLine int    `json:"first_diff_line,omitempty"`
}

func (a *app) aiRead(ctx context.Context, _ *mcp.CallToolRequest, in aiReadArgs) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	if err := checkAIName(in.Name); err != nil {
		return fail(err)
	}
	var local []byte
	if in.File != "" {
		var err error
		if local, err = os.ReadFile(in.File); err != nil {
			return fail(err)
		}
	}
	body, err := a.client.Post(ctx, "ai/read", map[string]string{"path": in.Name})
	if err != nil {
		return fail(err)
	}
	if in.Raw && in.File == "" {
		return raw(body)
	}
	code, mtime, err := summary.AIRead(body)
	if err != nil {
		return fail(err)
	}
	if in.File == "" {
		return ok(aiReadResult{Name: in.Name, Version: summary.AIVersion(code), Mtime: mtime, Lines: strings.Count(code, "\n") + 1, Code: code})
	}
	return ok(aiCompareResult{
		Name:          in.Name,
		File:          in.File,
		Identical:     code == string(local),
		OnlineVersion: summary.AIVersion(code),
		LocalVersion:  summary.AIVersion(string(local)),
		FirstDiffLine: summary.FirstDiffLine(code, string(local)),
	})
}

func checkAIName(name string) error {
	if name == "" {
		return fmt.Errorf("nom d'IA vide")
	}
	if strings.HasSuffix(name, leekExt) {
		return fmt.Errorf("nom d'IA %q : les IA en ligne n'ont pas l'extension .leek", name)
	}
	return nil
}

type aiPushArgs struct {
	Files []string `json:"files" jsonschema:"fichiers .leek locaux ; chacun est poussé dans l'IA en ligne de même nom sans .leek, dans l'ordre _grp, _nasu, puis le reste"`
}

// Statuts d'un fichier poussé.
const (
	pushPushed    = "pushed"    // écrit et relu identique
	pushUnchanged = "unchanged" // déjà identique en ligne, rien écrit
	pushProblems  = "problems"  // écrit mais refusé à la compilation
	pushMismatch  = "mismatch"  // écrit mais relu différent
	pushSkipped   = "skipped"   // non poussé : un fichier précédent a échoué
)

type aiPushFile struct {
	Name             string              `json:"name"`
	File             string              `json:"file"`
	Status           string              `json:"status"`
	OldVersion       string              `json:"old_version,omitempty"`
	NewVersion       string              `json:"new_version,omitempty"`
	VersionUnchanged bool                `json:"version_unchanged,omitempty"`
	Includers        map[string]bool     `json:"includers,omitempty"`
	PlayedBy         []string            `json:"played_by,omitempty"`
	Problems         []summary.AIProblem `json:"problems,omitempty"`
	FirstDiffLine    int                 `json:"first_diff_line,omitempty"`

	code []byte
}

type aiPushResult struct {
	Files    []*aiPushFile `json:"files"`
	Warnings []string      `json:"warnings,omitempty"`
}

// pushRank ordonne un lot : les _grp, puis les _nasu (inclus après le _grp),
// puis les IA qui les incluent.
func pushRank(name string) int {
	switch {
	case strings.HasSuffix(name, "_grp"):
		return 0
	case strings.HasSuffix(name, "_nasu"):
		return 1
	}
	return 2
}

func (a *app) aiPush(ctx context.Context, _ *mcp.CallToolRequest, in aiPushArgs) (*mcp.CallToolResult, any, error) {
	if !a.client.HasToken() {
		return fail(errNoToken)
	}
	if len(in.Files) == 0 {
		return fail(fmt.Errorf("aucun fichier à pousser"))
	}
	// Tout le lot est validé avant la première écriture.
	files := make([]*aiPushFile, 0, len(in.Files))
	seen := map[string]string{}
	for _, path := range in.Files {
		base := filepath.Base(path)
		if !strings.HasSuffix(base, leekExt) {
			return fail(fmt.Errorf("%s : fichier .leek attendu", path))
		}
		name := strings.TrimSuffix(base, leekExt)
		if err := checkAIName(name); err != nil {
			return fail(fmt.Errorf("%s : %w", path, err))
		}
		if prev, dup := seen[name]; dup {
			return fail(fmt.Errorf("IA %q en double : %s et %s", name, prev, path))
		}
		seen[name] = path
		code, err := os.ReadFile(path)
		if err != nil {
			return fail(err)
		}
		files = append(files, &aiPushFile{Name: name, File: path, NewVersion: summary.AIVersion(string(code)), code: code})
	}
	sort.SliceStable(files, func(i, j int) bool { return pushRank(files[i].Name) < pushRank(files[j].Name) })

	treeBody, err := a.client.Get(ctx, "ai/get-farmer-tree")
	if err != nil {
		return fail(err)
	}
	tree, err := summary.AITree(treeBody)
	if err != nil {
		return fail(err)
	}
	for _, f := range files {
		if !tree.Has(f.Name) {
			return fail(fmt.Errorf("IA %q absente en ligne : la créer sur le site d'abord", f.Name))
		}
		f.PlayedBy = tree.PlayedBy(f.Name)
	}

	res := aiPushResult{Files: files}
	failed := false
	for _, f := range files {
		if failed {
			f.Status = pushSkipped
			continue
		}
		if err := a.pushOne(ctx, f); err != nil {
			// Erreur réseau ou d'API : les fichiers déjà traités restent dans le résultat.
			f.Status = "error: " + err.Error()
		}
		switch f.Status {
		case pushPushed, pushUnchanged:
		default:
			failed = true
		}
		if f.VersionUnchanged {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s : code modifié mais VERSION inchangée (%q)", f.Name, f.OldVersion))
		}
	}
	out, _, _ := ok(res)
	out.IsError = failed
	return out, nil, nil
}

// pushOne lit la version en ligne, écrit le fichier s'il diffère, vérifie les
// problems et relit le code écrit.
func (a *app) pushOne(ctx context.Context, f *aiPushFile) error {
	body, err := a.client.Post(ctx, "ai/read", map[string]string{"path": f.Name})
	if err != nil {
		return err
	}
	online, _, err := summary.AIRead(body)
	if err != nil {
		return err
	}
	f.OldVersion = summary.AIVersion(online)
	if online == string(f.code) {
		f.Status = pushUnchanged
		return nil
	}
	f.VersionUnchanged = f.OldVersion == f.NewVersion
	body, err = a.client.Post(ctx, "ai/write", map[string]string{"path": f.Name, "code": string(f.code)})
	if err != nil {
		return err
	}
	w, err := summary.AIWrite(body, f.Name)
	if err != nil {
		return err
	}
	f.Includers = w.Includers
	if len(w.Problems) > 0 {
		f.Status, f.Problems = pushProblems, w.Problems
		return nil
	}
	body, err = a.client.Post(ctx, "ai/read", map[string]string{"path": f.Name})
	if err != nil {
		return err
	}
	written, _, err := summary.AIRead(body)
	if err != nil {
		return err
	}
	if written != string(f.code) {
		f.Status, f.FirstDiffLine = pushMismatch, summary.FirstDiffLine(written, string(f.code))
		return nil
	}
	f.Status = pushPushed
	return nil
}
