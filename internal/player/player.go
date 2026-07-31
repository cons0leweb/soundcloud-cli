package player

import (
	"errors"
	"io"
	"os/exec"
	"strconv"
	"sync"
)

// Player owns one headless ffplay process at a time.
type Player struct {
	mu     sync.Mutex
	binary string
	cmd    *exec.Cmd
	input  io.WriteCloser
	paused bool
	volume int
	muted  bool
}

func New(binary string) *Player {
	return &Player{binary: binary, volume: 80}
}

// Play replaces the current stream and returns a channel that reports its end.
func (p *Player) Play(streamURL string) (<-chan error, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stopLocked()
	cmd := exec.Command(p.binary,
		"-nodisp",
		"-autoexit",
		"-loglevel", "quiet",
		"-volume", strconv.Itoa(p.volume),
		streamURL,
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		_ = input.Close()
		return nil, err
	}

	p.cmd = cmd
	p.input = input
	p.paused = false
	p.muted = false
	done := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		if p.cmd == cmd {
			p.cmd = nil
			p.input = nil
			p.paused = false
			p.muted = false
		}
		p.mu.Unlock()
		done <- err
		close(done)
	}()
	return done, nil
}

// TogglePause sends ffplay its portable interactive pause command.
func (p *Player) TogglePause() (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd == nil || p.cmd.Process == nil || p.input == nil {
		return false, errors.New("nothing is playing")
	}
	if err := togglePause(p.cmd, p.input, p.paused); err != nil {
		return p.paused, err
	}
	p.paused = !p.paused
	return p.paused, nil
}

// AdjustVolume changes ffplay volume in five-point steps.
func (p *Player) AdjustVolume(delta int) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd == nil || p.input == nil {
		return p.volume, errors.New("nothing is playing")
	}
	target := max(0, min(100, p.volume+delta))
	key := "0"
	if target < p.volume {
		key = "9"
	}
	steps := (abs(target-p.volume) + 4) / 5
	for range steps {
		if err := p.sendKeyLocked(key); err != nil {
			return p.volume, err
		}
	}
	p.volume = target
	return p.volume, nil
}

// ToggleMute toggles ffplay audio without changing the stored volume.
func (p *Player) ToggleMute() (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd == nil || p.input == nil {
		return p.muted, errors.New("nothing is playing")
	}
	if err := p.sendKeyLocked("m"); err != nil {
		return p.muted, err
	}
	p.muted = !p.muted
	return p.muted, nil
}

func (p *Player) sendKeyLocked(key string) error {
	_, err := io.WriteString(p.input, key)
	return err
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
}

func (p *Player) Close() error {
	p.Stop()
	return nil
}

func (p *Player) stopLocked() {
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	if p.input != nil {
		_ = p.input.Close()
	}
	p.cmd = nil
	p.input = nil
	p.paused = false
	p.muted = false
}
