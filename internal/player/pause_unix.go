//go:build unix

package player

import (
	"io"
	"os/exec"
	"syscall"
)

func togglePause(command *exec.Cmd, _ io.Writer, paused bool) error {
	signal := syscall.SIGSTOP
	if paused {
		signal = syscall.SIGCONT
	}
	return command.Process.Signal(signal)
}
