package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type colorRule struct {
	re    *regexp.Regexp
	style lipgloss.Style
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

// parseColorRule parses a "regex=fg" or "regex=fg/bg" spec into a colorRule.
func parseColorRule(spec string) (colorRule, error) {
	idx := strings.LastIndex(spec, "=")
	if idx < 0 {
		return colorRule{}, fmt.Errorf("expected regex=fg or regex=fg/bg, got %q", spec)
	}
	pattern, colorSpec := spec[:idx], spec[idx+1:]
	re, err := regexp.Compile(pattern)
	if err != nil {
		return colorRule{}, fmt.Errorf("invalid regex %q: %w", pattern, err)
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
	return colorRule{re: re, style: style}, nil
}

// render applies the first matching rule's style to line, if any.
func render(line string, rules []colorRule) string {
	for _, r := range rules {
		if r.re.MatchString(line) {
			return r.style.Render(line)
		}
	}
	return line
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
