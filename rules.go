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
	c, ok := namedColors[strings.ToLower(name)]
	if !ok {
		c = namedColors["yellow"]
	}
	return lipgloss.NewStyle().Foreground(c).Bold(true)
}

// parseColorRule parses a "regex=color" spec into a colorRule.
func parseColorRule(spec string) (colorRule, error) {
	idx := strings.LastIndex(spec, "=")
	if idx < 0 {
		return colorRule{}, fmt.Errorf("expected regex=color, got %q", spec)
	}
	pattern, colorName := spec[:idx], spec[idx+1:]
	re, err := regexp.Compile(pattern)
	if err != nil {
		return colorRule{}, fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	if _, ok := namedColors[strings.ToLower(colorName)]; !ok {
		return colorRule{}, fmt.Errorf("unknown color %q (want one of red, green, yellow, blue, magenta, cyan, white)", colorName)
	}
	return colorRule{re: re, style: namedStyle(colorName)}, nil
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
