package player

import (
	"errors"
	"io"
	"os/exec"
	"sync"
)

// Player owns one headless ffplay process at a time.
type Player struct {
	mu     sync.Mutex
	binary string
	cmd    *exec.Cmd
	input  io.WriteCloser
	paused bool
}

func New(binary string) *Player {
	return &Player{binary: binary}
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
		"-volume", "80",
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
	done := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		if p.cmd == cmd {
			p.cmd = nil
			p.input = nil
			p.paused = false
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
	if _, err := io.WriteString(p.input, "p"); err != nil {
		return p.paused, err
	}
	p.paused = !p.paused
	return p.paused, nil
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
}
