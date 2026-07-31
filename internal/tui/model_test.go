package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSearchInputPreservesSpaces(t *testing.T) {
	model := New(nil, nil)
	model.query = []rune("deep")
	updated, _ := model.handleKey(tea.KeyMsg{Type: tea.KeySpace})
	got := updated.(Model)
	if string(got.query) != "deep " {
		t.Fatalf("query = %q, want %q", string(got.query), "deep ")
	}
}
