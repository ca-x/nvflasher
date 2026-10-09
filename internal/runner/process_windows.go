//go:build windows

package runner

import "os/exec"

func prepare(cmd *exec.Cmd) {}
func terminate(cmd *exec.Cmd, finished <-chan struct{}) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
