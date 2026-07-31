//go:build linux

package player

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestTogglePauseSuspendsProcess(t *testing.T) {
	command := exec.Command("sleep", "10")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	}()

	if err := togglePause(command, nil, false); err != nil {
		t.Fatal(err)
	}
	waitForProcessState(t, command.Process.Pid, "T")

	if err := togglePause(command, nil, true); err != nil {
		t.Fatal(err)
	}
	waitForProcessState(t, command.Process.Pid, "S")
}

func waitForProcessState(t *testing.T, pid int, want string) {
	t.Helper()
	path := fmt.Sprintf("/proc/%d/status", pid)
	for range 50 {
		data, err := os.ReadFile(path)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "State:") && strings.Contains(line, "\t"+want) {
					return
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("process %d did not reach state %s", pid, want)
}
