package summary

import "testing"

func TestAITree(t *testing.T) {
	s, err := AITree(fixture(t, "ai_tree.json"))
	if err != nil {
		t.Fatal(err)
	}
	if s.LeekAIs["135146"] != "force" || s.LeekAIs["135231"] != "paladin" {
		t.Fatalf("leek_ais : %v", s.LeekAIs)
	}
	if !s.Has("force_grp") || s.Has("force_grp.leek") {
		t.Fatal("Has")
	}
	for i := 1; i < len(s.Files); i++ {
		if s.Files[i-1].Path > s.Files[i].Path {
			t.Fatalf("fichiers non triés : %v", s.Files)
		}
	}
	if got := s.PlayedBy("maje"); len(got) != 1 || got[0] != "135204" {
		t.Fatalf("PlayedBy(maje) = %v", got)
	}
	if got := s.PlayedBy("maje_grp"); len(got) != 0 {
		t.Fatalf("PlayedBy(maje_grp) = %v", got)
	}
}

func TestAIVersion(t *testing.T) {
	code, _, err := AIRead(fixture(t, "ai_read_force_nasu.json"))
	if err != nil {
		t.Fatal(err)
	}
	if v := AIVersion(code); v == "" || v[:2] != "N-" {
		t.Fatalf("n_VERSION de force_nasu : %q", v)
	}
	cases := map[string]string{
		"global VERSION = \"2026-10-06i\";   // tour 1": "2026-10-06i",
		"// x\nglobal g_VERSION = \"G23-a\";":           "G23-a",
		"var VERSION = \"x\";":                          "",
		"// global VERSION = \"x\";":                    "",
		"":                                              "",
	}
	for code, want := range cases {
		if got := AIVersion(code); got != want {
			t.Errorf("AIVersion(%q) = %q, attendu %q", code, got, want)
		}
	}
}

func TestAIWrite(t *testing.T) {
	r, err := AIWrite(fixture(t, "ai_write_force_grp.json"), "force_grp")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Problems) != 0 || !r.Includers["force"] || len(r.Includers) != 1 {
		t.Fatalf("force_grp : %+v", r)
	}
	r, err = AIWrite(fixture(t, "ai_write_paladin_nasu_problems.json"), "paladin_nasu")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Problems) != 3 {
		t.Fatalf("problems : %+v", r.Problems)
	}
	p := r.Problems[0]
	if p.File != "" || p.Line != 27 || p.Col != 1 || p.Code != 33 || len(p.Params) != 1 || p.Params[0] != "g_FOCUS_ORDER" {
		t.Fatalf("premier problem : %+v", p)
	}
	if _, err := AIWrite(fixture(t, "ai_write_force.json"), "force_grp"); err == nil {
		t.Fatal("entrée absente : erreur attendue")
	}
}

func TestFirstDiffLine(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"a\nb", "a\nb", 0},
		{"a\nb", "a\nc", 2},
		{"a", "a\nb", 2},
		{"a\nb\n", "a\nb", 3},
	}
	for _, c := range cases {
		if got := FirstDiffLine(c.a, c.b); got != c.want {
			t.Errorf("FirstDiffLine(%q, %q) = %d, attendu %d", c.a, c.b, got, c.want)
		}
	}
}
