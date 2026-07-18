package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type colorRule struct {
	re        *regexp.Regexp
	style     lipgloss.Style
	wholeLine bool   // true: color the entire line; false: color only the matched text
	spec      string // the original "regex=fg[/bg][:scope]" text, kept for display in the rules panel
}

var namedColors = map[string]lipgloss.Color{
	"red":     lipgloss.Color("9"),
	"green":   lipgloss.Color("10"),
	"yellow":  lipgloss.Color("11"),
	"blue":    lipgloss.Color("12"),
	"magenta": lipgloss.Color("13"),
	"cyan":    lipgloss.Color("14"),
	"white":   lipgloss.Color("15"),
}

func namedStyle(name string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colorOrYellow(name)).Bold(true)
}

func colorOrYellow(name string) lipgloss.Color {
	if c, ok := namedColors[strings.ToLower(name)]; ok {
		return c
	}
	return namedColors["yellow"]
}

// parseColorRule parses a "regex=fg[/bg][:scope]" spec into a colorRule.
// scope is "line" (default, colors the whole line) or "word" (colors only
// the matched text).
func parseColorRule(spec string) (colorRule, error) {
	idx := strings.LastIndex(spec, "=")
	if idx < 0 {
		return colorRule{}, fmt.Errorf("expected regex=fg[/bg][:scope], got %q", spec)
	}
	pattern, rest := spec[:idx], spec[idx+1:]
	re, err := regexp.Compile(pattern)
	if err != nil {
		return colorRule{}, fmt.Errorf("invalid regex %q: %w", pattern, err)
	}

	colorSpec, scope, hasScope := strings.Cut(rest, ":")
	wholeLine := true
	if hasScope {
		switch scope {
		case "word", "match":
			wholeLine = false
		case "line":
			wholeLine = true
		default:
			return colorRule{}, fmt.Errorf("unknown scope %q (want \"word\" or \"line\")", scope)
		}
	}

	fgName, bgName, hasBg := strings.Cut(colorSpec, "/")
	fg, ok := namedColors[strings.ToLower(fgName)]
	if !ok {
		return colorRule{}, fmt.Errorf("unknown color %q (want one of red, green, yellow, blue, magenta, cyan, white)", fgName)
	}
	style := lipgloss.NewStyle().Foreground(fg).Bold(true)
	if hasBg {
		bg, ok := namedColors[strings.ToLower(bgName)]
		if !ok {
			return colorRule{}, fmt.Errorf("unknown background color %q (want one of red, green, yellow, blue, magenta, cyan, white)", bgName)
		}
		style = style.Background(bg)
	}
	return colorRule{re: re, style: style, wholeLine: wholeLine, spec: spec}, nil
}

// render colors line according to rules: whole-line rules are checked first
// and the first one that matches wins the entire line; otherwise, word-scope
// rules color just their matched spans (earlier rules and earlier matches
// take priority on overlap).
func render(line string, rules []colorRule) string {
	for _, r := range rules {
		if r.wholeLine && r.re.MatchString(line) {
			return r.style.Render(line)
		}
	}

	type span struct {
		start, end int
		style      lipgloss.Style
	}
	var spans []span
	for _, r := range rules {
		if r.wholeLine {
			continue
		}
		for _, idx := range r.re.FindAllStringIndex(line, -1) {
			spans = append(spans, span{idx[0], idx[1], r.style})
		}
	}
	if len(spans) == 0 {
		return line
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })

	var b strings.Builder
	pos := 0
	for _, sp := range spans {
		if sp.start < pos {
			continue // overlaps an already-placed span
		}
		b.WriteString(line[pos:sp.start])
		b.WriteString(sp.style.Render(line[sp.start:sp.end]))
		pos = sp.end
	}
	b.WriteString(line[pos:])
	return b.String()
}

// visible reports whether line should be displayed given the static
// hide/show rules: any match against hideRules hides the line, and if
// showRules is non-empty the line must match at least one of them.
func visible(line string, hideRules, showRules []*regexp.Regexp) bool {
	for _, re := range hideRules {
		if re.MatchString(line) {
			return false
		}
	}
	if len(showRules) == 0 {
		return true
	}
	for _, re := range showRules {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}
