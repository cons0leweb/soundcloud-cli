package tui

import (
	"strings"
	"testing"
)

func TestFormatDuration(t *testing.T) {
	tests := map[int]string{
		0:    "0:00",
		65:   "1:05",
		3661: "1:01:01",
		-1:   "0:00",
	}
	for seconds, want := range tests {
		if got := formatDuration(seconds); got != want {
			t.Errorf("formatDuration(%d) = %q, want %q", seconds, got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("музыка", 4); got != "муз…" {
		t.Fatalf("truncate unicode = %q", got)
	}
	if got := truncate("ok", 5); got != "ok" {
		t.Fatalf("truncate short string = %q", got)
	}
}

func TestCompactPlayerUsesProgressBarWhenWaveformHidden(t *testing.T) {
	model := New(nil, nil)
	model.width = 80
	model.hasCurrent = true
	model.currentTrack.Title = "Track"
	model.currentTrack.Duration = 120
	model.waveformVisible = false
	player := model.renderPlayer(80)
	if strings.Count(player, "\n") != 3 || !strings.Contains(player, "─") {
		t.Fatalf("unexpected compact player: %q", player)
	}
}
