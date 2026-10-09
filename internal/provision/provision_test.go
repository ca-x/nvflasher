package provision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectHostSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "etc"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "machine-id"), filepath.Join(root, "etc/machine-id")); err != nil {
		t.Fatal(err)
	}
	if err := safeTarget(root, filepath.Join(root, "etc/machine-id")); err == nil {
		t.Fatal("host symlink accepted")
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := safeTarget(root, filepath.Join(root, "escape", "file")); err == nil {
		t.Fatal("escaped parent accepted")
	}
}

func TestSteps(t *testing.T) {
	steps, err := Steps("/tmp/L4T", Options{Username: "dev", Hostname: "jetson", Password: "sensitive-password", Packages: "curl htop", Script: "echo ok"})
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 3 {
		t.Fatalf("got %d steps", len(steps))
	}
	args := strings.Join(steps[0].Command.Args, " ")
	if !strings.Contains(args, "--accept-license") || !strings.Contains(steps[1].Command.Input, "DEBIAN_FRONTEND=noninteractive") || steps[2].Command.Args[2] != "/bin/bash" {
		t.Fatalf("invalid steps: %+v", steps)
	}
	_, err = Steps("/tmp", Options{Username: "foo;rm", Hostname: "jetson", Password: "pass"})
	if err == nil {
		t.Fatal("invalid username accepted")
	}
}
func TestPassword(t *testing.T) {
	one, err := GeneratePassword()
	if err != nil || len(one) != 12 {
		t.Fatal(one, err)
	}
	two, err := GeneratePassword()
	if err != nil || one == two {
		t.Fatal("password not random", err)
	}
}
