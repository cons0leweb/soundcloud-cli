package tui

import (
	"testing"

	"github.com/cons0leweb/soundcloud-cli/internal/soundcloud"
)

func TestQueueSkipsCollectionsAndAdvances(t *testing.T) {
	queue := newPlaybackQueue()
	tracks := []soundcloud.Track{
		{Title: "Set", URL: "set", Collection: true},
		{Title: "One", URL: "one"},
		{Title: "Two", URL: "two"},
	}
	if got := queue.replace(tracks, "one"); got != 0 {
		t.Fatalf("selected index = %d, want 0", got)
	}
	if got := queue.next(false); got != 1 {
		t.Fatalf("next index = %d, want 1", got)
	}
	queue.index = 1
	if got := queue.next(false); got != -1 {
		t.Fatalf("queue without repeat wrapped to %d", got)
	}
}

func TestQueueRepeatModes(t *testing.T) {
	queue := newPlaybackQueue()
	queue.tracks = []soundcloud.Track{{URL: "one"}, {URL: "two"}}
	queue.index = 1
	queue.repeat = repeatAll
	if got := queue.next(false); got != 0 {
		t.Fatalf("repeat all = %d, want 0", got)
	}
	queue.repeat = repeatOne
	if got := queue.next(false); got != 1 {
		t.Fatalf("repeat one = %d, want 1", got)
	}
}
