package tui

import (
	"slices"
	"testing"

	"github.com/cons0leweb/soundcloud-cli/internal/soundcloud"
)

func TestWaveformIsStablePerTrackAndMoves(t *testing.T) {
	track := soundcloud.Track{Title: "Night Drive", Artist: "Test", URL: "https://soundcloud.com/test/night-drive"}
	first := buildWaveform(track, 48, 0)
	repeated := buildWaveform(track, 48, 0)
	moved := buildWaveform(track, 48, 5)
	if !slices.Equal(first.top, repeated.top) || !slices.Equal(first.bottom, repeated.bottom) {
		t.Fatal("same track and frame produced a different waveform")
	}
	if slices.Equal(first.top, moved.top) && slices.Equal(first.bottom, moved.bottom) {
		t.Fatal("animated frame did not move the waveform")
	}
	if len(first.top) != 48 || len(first.bottom) != 48 {
		t.Fatalf("waveform width = %d/%d, want 48", len(first.top), len(first.bottom))
	}
}

func TestWaveformDiffersBetweenTracks(t *testing.T) {
	left := buildWaveform(soundcloud.Track{URL: "https://soundcloud.com/a/one"}, 32, 0)
	right := buildWaveform(soundcloud.Track{URL: "https://soundcloud.com/b/two"}, 32, 0)
	if slices.Equal(left.bottom, right.bottom) && slices.Equal(left.top, right.top) {
		t.Fatal("different tracks produced identical waveforms")
	}
}
