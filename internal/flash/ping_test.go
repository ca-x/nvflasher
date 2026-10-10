package flash

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPing6AdapterForwardsArgumentsAndExitStatus(t *testing.T) {
	dir := t.TempDir()
	ping := filepath.Join(dir, "ping with ' quote")
	if err := os.WriteFile(ping, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\nexit 7\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := writePing6Adapter(dir, ping); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(filepath.Join(dir, "ping6"), "-c", "1", "fe80::1%enp0s20f0u1")
	output, err := cmd.CombinedOutput()
	if string(output) != "-6\n-c\n1\nfe80::1%enp0s20f0u1\n" {
		t.Fatalf("arguments: %q", output)
	}
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 7 {
		t.Fatalf("exit status: %v", err)
	}
}
