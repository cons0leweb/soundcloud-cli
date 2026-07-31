//go:build !unix

package player

import (
	"io"
	"os/exec"
)

func togglePause(_ *exec.Cmd, input io.Writer, _ bool) error {
	_, err := io.WriteString(input, "p")
	return err
}
