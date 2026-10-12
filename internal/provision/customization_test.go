package provision

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCustomizationValidation(t *testing.T) {
	for _, o := range []Options{{Suspend: "yes"}, {ScreenLock: "false"}, {ScreenBlank: "off"}, {Autostart: "relative"}, {Autostart: "/opt/app%f"}, {Autostart: "/opt/../app"}, {DebFiles: "relative.deb"}} {
		if validateCustomization(o) == nil {
			t.Fatalf("accepted %+v", o)
		}
	}
	if err := validateCustomization(Options{Suspend: "enable", ScreenLock: "disable", Autostart: "/opt/my app", DebFiles: "/tmp/a.deb\n/tmp/b.deb"}); err != nil {
		t.Fatal(err)
	}
	if desktopConfiguration(Options{}) != "" {
		t.Fatal("empty settings not a no-op")
	}
}
func TestCustomizationSteps(t *testing.T) {
	o := Options{Username: "dev", Hostname: "jetson", Password: "test", AptMirror: "https://ports.ubuntu.com/ubuntu-ports", ChineseSupport: true, ScreenLock: "disable", DebFiles: "/tmp/a.deb", Autostart: "/opt/app"}
	steps, err := Steps("/tmp/l4t", o)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{"Create default user", "Configure chroot system", "Install packages", "Install local DEBs", "Configure desktop defaults"}
	if len(steps) != len(names) {
		t.Fatalf("steps: %+v", steps)
	}
	for i, step := range steps {
		if step.Name != names[i] {
			t.Fatalf("step %d: %s", i, step.Name)
		}
		if step.Command.Input != "" {
			cmd := exec.Command("bash", "-n")
			cmd.Stdin = strings.NewReader(step.Command.Input)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("shell syntax: %v %s", err, out)
			}
		}
	}
	if strings.Contains(steps[2].Command.Input, "update-locale") {
		t.Fatal("Chinese support forces locale")
	}
}
func TestDesktopFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "opt"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "opt/app"), []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	o := Options{Username: "dev", Suspend: "disable", ScreenBlank: "enable", ScreenLock: "disable", ChineseSupport: true, Autostart: "/opt/app"}
	for i := 0; i < 2; i++ {
		if err := installDesktop(root, o); err != nil {
			t.Fatal(err)
		}
	}
	// Changing one setting must not undo an earlier setting left blank.
	lockPath := filepath.Join(root, "etc/dconf/db/nvflasher.d/11-screen-lock")
	before, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := installDesktop(root, Options{ScreenBlank: "disable"}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(lockPath)
	if err != nil || string(before) != string(after) {
		t.Fatal("blank lock option changed previous defaults", err)
	}
	profile, _ := os.ReadFile(filepath.Join(root, "etc/dconf/profile/user"))
	if strings.Count(string(profile), "system-db:nvflasher") != 1 {
		t.Fatal(string(profile))
	}
	data, err := os.ReadFile(filepath.Join(root, "home/dev/.config/autostart/nvflasher.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Exec=\"/opt/app\"") {
		t.Fatal(string(data))
	}
	outside := t.TempDir()
	os.MkdirAll(filepath.Join(root, "home/evil"), 0755)
	os.Symlink(outside, filepath.Join(root, "home/evil/.config"))
	o.Username = "evil"
	if installDesktop(root, o) == nil {
		t.Fatal("accepted symlink parent")
	}
}
func TestDebArchitectures(t *testing.T) {
	if _, err := exec.LookPath("dpkg-deb"); err != nil {
		t.Skip("dpkg-deb unavailable")
	}
	for _, arch := range []string{"arm64", "all", "amd64"} {
		t.Run(arch, func(t *testing.T) {
			dir := t.TempDir()
			os.MkdirAll(filepath.Join(dir, "pkg/DEBIAN"), 0755)
			os.WriteFile(filepath.Join(dir, "pkg/DEBIAN/control"), []byte("Package: test-package\nVersion: 1.0\nArchitecture: "+arch+"\nMaintainer: Test <test@example.com>\nDescription: Test\n"), 0644)
			path := filepath.Join(dir, "test.deb")
			if out, err := exec.Command("dpkg-deb", "--build", filepath.Join(dir, "pkg"), path).CombinedOutput(); err != nil {
				t.Fatalf("%v %s", err, out)
			}
			err := checkDeb(context.Background(), path)
			if (err == nil) != (arch != "amd64") {
				t.Fatalf("architecture %s: %v", arch, err)
			}
			if arch == "arm64" {
				root := t.TempDir()
				stage, err := stageDebFiles(context.Background(), root, path)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(filepath.Join(stage, "package-0.deb")); err != nil {
					t.Fatal(err)
				}
				root = t.TempDir()
				os.Symlink(dir, filepath.Join(root, "tmp"))
				if _, err := stageDebFiles(context.Background(), root, path); err == nil {
					t.Fatal("accepted symlink staging directory")
				}
			}
		})
	}
}
