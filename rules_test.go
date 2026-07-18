package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestMain(m *testing.M) {
	// Color detection looks at the terminal; force it on so style.Render
	// actually emits ANSI codes under `go test`, which has no TTY.
	lipgloss.SetColorProfile(termenv.ANSI)
	os.Exit(m.Run())
}

func TestParseColorRuleScope(t *testing.T) {
	cases := []struct {
		name      string
		spec      string
		wantErr   bool
		wholeLine bool
	}{
		{name: "default is whole line", spec: "ERROR=red", wholeLine: true},
		{name: "explicit line", spec: "ERROR=red:line", wholeLine: true},
		{name: "word scope", spec: "ERROR=red:word", wholeLine: false},
		{name: "match scope alias", spec: "ERROR=red:match", wholeLine: false},
		{name: "fg and bg", spec: "ERROR=white/red", wholeLine: true},
		{name: "fg, bg and scope", spec: "ERROR=white/red:word", wholeLine: false},
		{name: "colon inside pattern is not the scope", spec: `\d{2}:\d{2}=cyan`, wholeLine: true},
		{name: "bad color", spec: "ERROR=notacolor", wantErr: true},
		{name: "bad bg", spec: "ERROR=red/notacolor", wantErr: true},
		{name: "bad scope", spec: "ERROR=red:paragraph", wantErr: true},
		{name: "missing =", spec: "ERROR", wantErr: true},
		{name: "bad regex", spec: "[=red", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rule, err := parseColorRule(tc.spec)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got none", tc.spec)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.spec, err)
			}
			if rule.wholeLine != tc.wholeLine {
				t.Fatalf("wholeLine = %v, want %v", rule.wholeLine, tc.wholeLine)
			}
		})
	}
}

func TestRenderWholeLine(t *testing.T) {
	rule, err := parseColorRule("ERROR=red")
	if err != nil {
		t.Fatal(err)
	}
	got := render("boot ERROR: sensor timeout", []colorRule{rule})
	if !strings.Contains(got, "boot ERROR: sensor timeout") {
		t.Fatalf("rendered output lost the original text: %q", got)
	}
	if got == "boot ERROR: sensor timeout" {
		t.Fatalf("expected styling to change the raw string, got it unchanged")
	}
	// The whole line is wrapped in one style, so there's exactly one reset
	// sequence at the very end, not one per word.
	if strings.Count(got, "\x1b[0m") > 1 {
		t.Fatalf("expected the whole line under one style, got multiple resets: %q", got)
	}
}

func TestRenderWordOnly(t *testing.T) {
	rule, err := parseColorRule("ERROR=red:word")
	if err != nil {
		t.Fatal(err)
	}
	line := "boot ERROR: sensor ERROR timeout"
	got := render(line, []colorRule{rule})

	plain := stripANSI(got)
	if plain != line {
		t.Fatalf("word-only rendering changed the text: got %q, want %q", plain, line)
	}
	if !strings.Contains(got, "boot ") || !strings.Contains(got, ": sensor ") {
		t.Fatalf("expected the non-matching parts to stay unstyled: %q", got)
	}
	if strings.Count(got, "ERROR") < 2 {
		t.Fatalf("expected both ERROR occurrences to be present: %q", got)
	}
}

func TestRenderLineRulePriorityOverWord(t *testing.T) {
	wordRule, err := parseColorRule("WARN=yellow:word")
	if err != nil {
		t.Fatal(err)
	}
	lineRule, err := parseColorRule("ERROR=red")
	if err != nil {
		t.Fatal(err)
	}
	// A whole-line match should win over a word-scope rule, regardless of order.
	got := render("ERROR and WARN both present", []colorRule{wordRule, lineRule})
	plain := stripANSI(got)
	if plain != "ERROR and WARN both present" {
		t.Fatalf("text corrupted: %q", plain)
	}
	if strings.Count(got, "\x1b[0m") != 1 {
		t.Fatalf("expected a single whole-line style to win, got %q", got)
	}
}

func TestVisible(t *testing.T) {
	hide := []*regexp.Regexp{regexp.MustCompile("DEBUG")}
	show := []*regexp.Regexp{regexp.MustCompile("ERROR|WARN")}

	cases := []struct {
		line string
		want bool
	}{
		{"plain heartbeat", false},   // no -show match
		{"DEBUG heartbeat", false},   // hidden even though nothing else applies
		{"ERROR: timeout", true},     // matches show
		{"DEBUG ERROR mixed", false}, // hide wins over show
	}
	for _, tc := range cases {
		if got := visible(tc.line, hide, show); got != tc.want {
			t.Errorf("visible(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func stripANSI(s string) string {
	re := regexp.MustCompile("\x1b\\[[0-9;]*m")
	return re.ReplaceAllString(s, "")
}
