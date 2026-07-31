package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cons0leweb/soundcloud-cli/internal/soundcloud"
)

type testPlayer struct{}

func (testPlayer) Play(string) (<-chan error, error) { return make(chan error), nil }
func (testPlayer) TogglePause() (bool, error)        { return false, nil }
func (testPlayer) AdjustVolume(int) (int, error)     { return 80, nil }
func (testPlayer) ToggleMute() (bool, error)         { return false, nil }
func (testPlayer) Stop()                             {}

func TestSearchInputPreservesSpaces(t *testing.T) {
	model := New(nil, nil)
	model.query = []rune("deep")
	updated, _ := model.handleKey(tea.KeyMsg{Type: tea.KeySpace})
	got := updated.(Model)
	if string(got.query) != "deep " {
		t.Fatalf("query = %q, want %q", string(got.query), "deep ")
	}
}

func TestStationResultStartsEndlessRadio(t *testing.T) {
	model := New(nil, testPlayer{})
	model.radioRequest = 7
	model.radioLoading = true
	seed := soundcloud.Track{Title: "Seed", URL: "https://soundcloud.com/a/seed"}
	related := []soundcloud.Track{
		{Title: "One", URL: "https://soundcloud.com/b/one"},
		{Title: "Two", URL: "https://soundcloud.com/c/two"},
	}

	updated, cmd := model.Update(stationResultMsg{seed: seed, tracks: related, request: 7, initial: true})
	got := updated.(Model)
	if cmd == nil || !got.radioMode || got.activeView != "radio" {
		t.Fatalf("radio did not start: mode=%v view=%q cmd=%v", got.radioMode, got.activeView, cmd)
	}
	if len(got.queue.tracks) != 3 || got.currentTrack.URL != seed.URL || got.playback != playbackLoading {
		t.Fatalf("unexpected radio queue/state: %#v current=%q state=%d", got.queue.tracks, got.currentTrack.URL, got.playback)
	}
}

func TestStationRefillResumesAtFirstNewTrack(t *testing.T) {
	model := New(nil, testPlayer{})
	model.radioMode = true
	model.radioLoading = true
	model.radioResume = true
	model.radioRequest = 3
	model.queue.tracks = []soundcloud.Track{{Title: "Seed", URL: "https://soundcloud.com/a/seed"}}
	model.queue.index = 0
	next := soundcloud.Track{Title: "Next", URL: "https://soundcloud.com/b/next"}

	updated, cmd := model.Update(stationResultMsg{tracks: []soundcloud.Track{next}, request: 3})
	got := updated.(Model)
	if cmd == nil || got.radioResume || got.currentTrack.URL != next.URL || got.playback != playbackLoading {
		t.Fatalf("radio refill did not resume: resume=%v current=%q state=%d", got.radioResume, got.currentTrack.URL, got.playback)
	}
}

func TestStopCancelsPendingRadioResume(t *testing.T) {
	model := New(nil, testPlayer{})
	model.radioLoading = true
	model.radioResume = true
	model.radioRequest = 4
	model.stopPlayback("Stopped")
	if model.radioLoading || model.radioResume || model.radioRequest != 5 || model.playback != playbackStopped {
		t.Fatalf("pending radio was not cancelled: loading=%v resume=%v request=%d state=%d", model.radioLoading, model.radioResume, model.radioRequest, model.playback)
	}
}
