package tui

import (
	"hash/fnv"
	"math"

	"github.com/cons0leweb/soundcloud-cli/internal/soundcloud"
)

type waveformRows struct {
	top    []rune
	bottom []rune
}

func buildWaveform(track soundcloud.Track, width int, frame uint64) waveformRows {
	if width < 1 {
		return waveformRows{}
	}
	seed := waveformSeed(track)
	rows := waveformRows{top: make([]rune, width), bottom: make([]rune, width)}
	offset := float64(frame) * 0.42
	phase := float64(seed%10_000) / 997.0
	for index := range width {
		x := float64(index) + offset
		value := math.Abs(
			math.Sin(x*0.19+phase)*0.52 +
				math.Sin(x*0.071+phase*1.7)*0.31 +
				math.Sin(x*0.43+phase*0.37)*0.17,
		)
		level := min(4, max(1, int(math.Round(value*4))))
		switch level {
		case 1:
			rows.top[index], rows.bottom[index] = ' ', '▂'
		case 2:
			rows.top[index], rows.bottom[index] = ' ', '▄'
		case 3:
			rows.top[index], rows.bottom[index] = '▂', '█'
		default:
			rows.top[index], rows.bottom[index] = '▄', '█'
		}
	}
	return rows
}

func waveformSeed(track soundcloud.Track) uint64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(track.URL))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(track.Title))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(track.Artist))
	return hash.Sum64()
}
