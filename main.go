// Command serial-filter is a cross-platform serial console with regex
// highlighting, similar in spirit to minicom but focused on log viewing.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"go.bug.st/serial"
)

// stringSlice collects repeated occurrences of a flag, e.g. -color a -color b.
type stringSlice []string

func (s *stringSlice) String() string     { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

func main() {
	var (
		listPorts    = flag.Bool("list", false, "list available serial ports and exit")
		baud         = flag.Int("baud", 115200, "baud rate")
		filter       = flag.String("filter", "", "regex to highlight in yellow (shorthand for -color regex=yellow)")
		onlyMatching = flag.Bool("only-matching", false, "start with the view filtered to -filter matches only")
		colorFlags   stringSlice
		hideFlags    stringSlice
		showFlags    stringSlice
	)
	flag.Var(&colorFlags, "color", "regex=fg[/bg][:word] highlight rule, repeatable (colors: red, green, yellow, blue, magenta, cyan, white; scope defaults to the whole line, use :word to color only the match)")
	flag.Var(&hideFlags, "hide", "regex; lines matching any -hide rule are never displayed, repeatable")
	flag.Var(&showFlags, "show", "regex; if any -show rule is given, only lines matching at least one are displayed, repeatable")
	flag.Parse()

	if *listPorts {
		ports, err := serial.GetPortsList()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error listing ports:", err)
			os.Exit(1)
		}
		if len(ports) == 0 {
			fmt.Println("no serial ports found")
			return
		}
		for _, p := range ports {
			fmt.Println(p)
		}
		return
	}

	portName := flag.Arg(0)
	if portName == "" {
		fmt.Fprintln(os.Stderr, "usage: serial-filter [-baud N] [-color regex=color]... [-filter regex] [-only-matching] <port>")
		fmt.Fprintln(os.Stderr, `use "-" as <port> to read from stdin instead of a serial port (handy for testing without hardware)`)
		os.Exit(1)
	}

	var rules []colorRule
	if *filter != "" {
		re, err := regexp.Compile(*filter)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid -filter regex:", err)
			os.Exit(1)
		}
		rules = append(rules, colorRule{re: re, style: namedStyle("yellow"), wholeLine: true})
	}
	for _, spec := range colorFlags {
		r, err := parseColorRule(spec)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid -color:", err)
			os.Exit(1)
		}
		rules = append(rules, r)
	}
	if *onlyMatching && *filter == "" {
		fmt.Fprintln(os.Stderr, "-only-matching requires -filter")
		os.Exit(1)
	}

	compile := func(specs []string, flagName string) []*regexp.Regexp {
		var res []*regexp.Regexp
		for _, spec := range specs {
			re, err := regexp.Compile(spec)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid -%s regex %q: %v\n", flagName, spec, err)
				os.Exit(1)
			}
			res = append(res, re)
		}
		return res
	}
	hideRules := compile(hideFlags, "hide")
	showRules := compile(showFlags, "show")

	var reader io.Reader
	if portName == "-" {
		reader = os.Stdin
	} else {
		mode := &serial.Mode{BaudRate: *baud}
		port, err := serial.Open(portName, mode)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error opening port:", err)
			os.Exit(1)
		}
		defer port.Close()
		reader = port
	}

	initialFilter := ""
	if *onlyMatching {
		initialFilter = *filter
	}

	p := tea.NewProgram(newModel(portName, reader, rules, hideRules, showRules, initialFilter), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
