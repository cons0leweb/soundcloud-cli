package tui

import (
	"math/rand"

	"github.com/cons0leweb/soundcloud-cli/internal/soundcloud"
)

type repeatMode uint8

const (
	repeatOff repeatMode = iota
	repeatAll
	repeatOne
)

type playbackQueue struct {
	tracks  []soundcloud.Track
	index   int
	shuffle bool
	repeat  repeatMode
}

func newPlaybackQueue() playbackQueue {
	return playbackQueue{index: -1}
}

func (q *playbackQueue) replace(source []soundcloud.Track, selectedURL string) int {
	q.tracks = q.tracks[:0]
	q.index = -1
	for _, track := range source {
		if track.Collection {
			continue
		}
		q.tracks = append(q.tracks, track)
		if track.URL == selectedURL {
			q.index = len(q.tracks) - 1
		}
	}
	return q.index
}

func (q *playbackQueue) selectIndex(index int) (soundcloud.Track, bool) {
	if index < 0 || index >= len(q.tracks) {
		return soundcloud.Track{}, false
	}
	q.index = index
	return q.tracks[index], true
}

func (q *playbackQueue) appendUnique(source []soundcloud.Track) int {
	seen := make(map[string]bool, len(q.tracks))
	for _, track := range q.tracks {
		seen[track.URL] = true
	}
	first := -1
	for _, track := range source {
		if track.Collection || track.URL == "" || seen[track.URL] {
			continue
		}
		seen[track.URL] = true
		q.tracks = append(q.tracks, track)
		if first < 0 {
			first = len(q.tracks) - 1
		}
	}
	return first
}

func (q playbackQueue) remaining() int {
	if q.index < 0 {
		return len(q.tracks)
	}
	return max(0, len(q.tracks)-q.index-1)
}

func (q *playbackQueue) next(manual bool) int {
	if len(q.tracks) == 0 {
		return -1
	}
	if !manual && q.repeat == repeatOne && q.index >= 0 {
		return q.index
	}
	if q.shuffle && len(q.tracks) > 1 {
		next := rand.Intn(len(q.tracks) - 1)
		if next >= q.index {
			next++
		}
		return next
	}
	if q.index+1 < len(q.tracks) {
		return q.index + 1
	}
	if manual || q.repeat == repeatAll {
		return 0
	}
	return -1
}

func (q *playbackQueue) previous() int {
	if len(q.tracks) == 0 {
		return -1
	}
	if q.shuffle && len(q.tracks) > 1 {
		return q.next(true)
	}
	if q.index > 0 {
		return q.index - 1
	}
	return len(q.tracks) - 1
}

func (q *playbackQueue) cycleRepeat() repeatMode {
	q.repeat = (q.repeat + 1) % 3
	return q.repeat
}

func (q playbackQueue) position() (int, int) {
	if q.index < 0 || len(q.tracks) == 0 {
		return 0, len(q.tracks)
	}
	return q.index + 1, len(q.tracks)
}
