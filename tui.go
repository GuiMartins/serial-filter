package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type lineMsg string
type readErrMsg struct{ err error }

type model struct {
	portName  string
	rules     []colorRule
	hideRules []*regexp.Regexp
	showRules []*regexp.Regexp
	lines     chan string
	errs      chan error

	all       []string // full raw history, kept for export regardless of the active filter
	viewport  viewport.Model
	search    textinput.Model
	searching bool
	filter    string // active interactive search filter (regex if valid, else plain substring); "" shows everything that passes hide/show
	paused    bool
	status    string
	ready     bool
	quitting  bool

	configMode  bool
	configInput textinput.Model
	configErr   string
}

func newModel(portName string, r io.Reader, rules []colorRule, hideRules, showRules []*regexp.Regexp, initialFilter string) model {
	lines := make(chan string, 256)
	errs := make(chan error, 1)
	go readLoop(r, lines, errs)

	ti := textinput.New()
	ti.Placeholder = "regex or plain text..."
	ti.Prompt = "/"

	ci := textinput.New()
	ci.Placeholder = "regex=fg[/bg][:scope]  or  -N to remove rule N"
	ci.Prompt = "rule> "

	return model{
		portName:    portName,
		rules:       rules,
		hideRules:   hideRules,
		showRules:   showRules,
		lines:       lines,
		errs:        errs,
		search:      ti,
		filter:      initialFilter,
		configInput: ci,
	}
}

func readLoop(r io.Reader, lines chan<- string, errs chan<- error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		errs <- err
	}
	close(lines)
}

func waitForLine(lines <-chan string, errs <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case line, ok := <-lines:
			if !ok {
				return readErrMsg{err: io.EOF}
			}
			return lineMsg(line)
		case err := <-errs:
			return readErrMsg{err: err}
		}
	}
}

func (m model) Init() tea.Cmd {
	return waitForLine(m.lines, m.errs)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		const headerHeight, footerHeight = 1, 1
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight - footerHeight
		}
		m.search.Width = msg.Width - 2
		m.configInput.Width = msg.Width - 2
		m.viewport.SetContent(m.renderVisible())
		return m, nil

	case lineMsg:
		m.all = append(m.all, string(msg))
		if !m.paused {
			atBottom := m.viewport.AtBottom()
			m.viewport.SetContent(m.renderVisible())
			if atBottom {
				m.viewport.GotoBottom()
			}
		}
		return m, waitForLine(m.lines, m.errs)

	case readErrMsg:
		if msg.err == io.EOF {
			m.status = "input closed"
		} else {
			m.status = "read error: " + msg.err.Error()
		}
		return m, nil

	case tea.KeyMsg:
		if m.configMode {
			switch msg.String() {
			case "esc":
				m.configMode = false
				m.configInput.Blur()
				m.configInput.SetValue("")
				m.configErr = ""
				return m, nil
			case "enter":
				value := strings.TrimSpace(m.configInput.Value())
				if value == "" {
					return m, nil
				}
				if rest, ok := strings.CutPrefix(value, "-"); ok {
					n, err := strconv.Atoi(rest)
					if err != nil || n < 1 || n > len(m.rules) {
						m.configErr = fmt.Sprintf("no rule #%s", rest)
						return m, nil
					}
					m.rules = append(m.rules[:n-1], m.rules[n:]...)
				} else {
					rule, err := parseColorRule(value)
					if err != nil {
						m.configErr = err.Error()
						return m, nil
					}
					m.rules = append(m.rules, rule)
				}
				m.configErr = ""
				m.configInput.SetValue("")
				m.viewport.SetContent(m.renderVisible())
				return m, nil
			}
			var cmd tea.Cmd
			m.configInput, cmd = m.configInput.Update(msg)
			return m, cmd
		}

		if m.searching {
			switch msg.String() {
			case "enter":
				m.filter = m.search.Value()
				m.searching = false
				m.search.Blur()
				m.viewport.SetContent(m.renderVisible())
				m.viewport.GotoBottom()
				return m, nil
			case "esc":
				m.searching = false
				m.search.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "p":
			m.paused = !m.paused
			if !m.paused {
				m.viewport.SetContent(m.renderVisible())
				m.viewport.GotoBottom()
			}
			return m, nil
		case "/":
			m.searching = true
			m.search.SetValue(m.filter)
			m.search.CursorEnd()
			m.search.Focus()
			return m, textinput.Blink
		case "f":
			m.filter = ""
			m.viewport.SetContent(m.renderVisible())
			return m, nil
		case "c":
			m.configMode = true
			m.configErr = ""
			m.configInput.Focus()
			return m, textinput.Blink
		case "e":
			name, err := m.export()
			if err != nil {
				m.status = "export failed: " + err.Error()
			} else {
				m.status = "exported to " + name
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// renderVisible re-renders the full buffer, applying the active filter and
// color rules. The buffer itself (m.all) is untouched so export always
// writes the complete, unfiltered history.
func (m model) renderVisible() string {
	var re *regexp.Regexp
	if m.filter != "" {
		re, _ = regexp.Compile(m.filter) // invalid regex falls back to plain substring match below
	}
	var b strings.Builder
	for _, line := range m.all {
		if !visible(line, m.hideRules, m.showRules) {
			continue
		}
		if m.filter != "" {
			var matched bool
			if re != nil {
				matched = re.MatchString(line)
			} else {
				matched = strings.Contains(line, m.filter)
			}
			if !matched {
				continue
			}
		}
		b.WriteString(render(line, m.rules))
		b.WriteByte('\n')
	}
	return b.String()
}

func (m model) export() (string, error) {
	name := fmt.Sprintf("serial-filter-export-%s.log", time.Now().Format("20060102-150405"))
	f, err := os.Create(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, line := range m.all {
		if _, err := w.WriteString(line + "\n"); err != nil {
			return "", err
		}
	}
	return name, w.Flush()
}

var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	pausedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	helpStyle   = lipgloss.NewStyle().Faint(true)
)

func (m model) View() string {
	if !m.ready {
		return "initializing..."
	}
	if m.quitting {
		return ""
	}
	if m.configMode {
		return m.configView()
	}

	header := headerStyle.Render(fmt.Sprintf(" %s ", m.portName))
	if m.paused {
		header += " " + pausedStyle.Render("[PAUSED]")
	}
	if m.filter != "" {
		header += fmt.Sprintf("  filter: %q", m.filter)
	}

	var footer string
	if m.searching {
		footer = m.search.View()
	} else {
		help := "q quit | p pause/resume | / search history | f clear filter | c color rules | e export log"
		if m.status != "" {
			help = m.status + "  |  " + help
		}
		footer = helpStyle.Render(help)
	}

	return fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), footer)
}

var errStyle = lipgloss.NewStyle().Foreground(namedColors["red"])

func (m model) configView() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render(" Color rules ") + "\n\n")

	if len(m.rules) == 0 {
		b.WriteString(helpStyle.Render("  (none yet)") + "\n")
	}
	for i, r := range m.rules {
		b.WriteString(fmt.Sprintf(" %d. %s\n", i+1, r.style.Render(r.spec)))
	}

	b.WriteString("\n")
	b.WriteString(m.configInput.View())
	b.WriteString("\n")
	if m.configErr != "" {
		b.WriteString(errStyle.Render("  "+m.configErr) + "\n")
	}
	b.WriteString(helpStyle.Render("  enter regex=fg[/bg][:scope] to add a rule, -N to remove rule N, Esc to close"))
	return b.String()
}
