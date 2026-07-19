package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func typeString(m model, s string) model {
	for _, r := range s {
		next, _ := m.Update(key(string(r)))
		m = next.(model)
	}
	return m
}

func newTestModel() model {
	m := newModel("TEST", strings.NewReader(""), nil, nil, nil, "")
	m.ready = true
	return m
}

func TestConfigPanelAddRule(t *testing.T) {
	m := newTestModel()

	next, _ := m.Update(key("c"))
	m = next.(model)
	if !m.configMode {
		t.Fatal("expected 'c' to enter config mode")
	}

	m = typeString(m, "ERROR=red:word")
	next, _ = m.Update(key("enter"))
	m = next.(model)

	if len(m.rules) != 1 {
		t.Fatalf("expected 1 rule after adding, got %d", len(m.rules))
	}
	if m.rules[0].wholeLine {
		t.Fatalf("expected word-scope rule, got wholeLine=true")
	}
	if m.configInput.Value() != "" {
		t.Fatalf("expected input to clear after a successful add, got %q", m.configInput.Value())
	}
	if m.configErr != "" {
		t.Fatalf("expected no error, got %q", m.configErr)
	}
}

func TestConfigPanelRejectsInvalidRule(t *testing.T) {
	m := newTestModel()
	next, _ := m.Update(key("c"))
	m = next.(model)

	m = typeString(m, "ERROR=notacolor")
	next, _ = m.Update(key("enter"))
	m = next.(model)

	if len(m.rules) != 0 {
		t.Fatalf("expected the invalid rule to be rejected, got %d rules", len(m.rules))
	}
	if m.configErr == "" {
		t.Fatal("expected an error message for an invalid rule")
	}
}

func TestConfigPanelRemoveRule(t *testing.T) {
	m := newTestModel()
	r1, err := parseColorRule("ERROR=red")
	if err != nil {
		t.Fatal(err)
	}
	r2, err := parseColorRule("WARN=yellow")
	if err != nil {
		t.Fatal(err)
	}
	m.rules = []colorRule{r1, r2}

	next, _ := m.Update(key("c"))
	m = next.(model)
	m = typeString(m, "-1")
	next, _ = m.Update(key("enter"))
	m = next.(model)

	if len(m.rules) != 1 {
		t.Fatalf("expected 1 rule left after removing #1, got %d", len(m.rules))
	}
	if m.rules[0].spec != "WARN=yellow" {
		t.Fatalf("expected the remaining rule to be WARN=yellow, got %q", m.rules[0].spec)
	}
}

func TestConfigPanelRemoveOutOfRange(t *testing.T) {
	m := newTestModel()
	r1, err := parseColorRule("ERROR=red")
	if err != nil {
		t.Fatal(err)
	}
	m.rules = []colorRule{r1}

	next, _ := m.Update(key("c"))
	m = next.(model)
	m = typeString(m, "-5")
	next, _ = m.Update(key("enter"))
	m = next.(model)

	if len(m.rules) != 1 {
		t.Fatalf("expected the out-of-range removal to be a no-op, got %d rules", len(m.rules))
	}
	if m.configErr == "" {
		t.Fatal("expected an error for an out-of-range rule number")
	}
}

func TestConfigPanelEscCloses(t *testing.T) {
	m := newTestModel()
	next, _ := m.Update(key("c"))
	m = next.(model)
	next, _ = m.Update(key("esc"))
	m = next.(model)

	if m.configMode {
		t.Fatal("expected Esc to close the config panel")
	}
}
