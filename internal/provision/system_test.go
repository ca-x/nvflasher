package provision

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSystemConfigurationValidation(t *testing.T) {
	for _, o := range []Options{
		{Timezone: "../../etc"}, {Timezone: "Asia/Shanghai'; echo bad"},
		{Locale: "zh_CN;echo bad"}, {AptMirror: "https://mirror.example/ubuntu"},
		{AptMirror: "https://user:pass@mirror.example/ubuntu-ports"},
		{AptMirror: "https://mirror.example/ubuntu-ports?x=y"},
		{AptMirror: "https://mirror.example/ubuntu-ports#frag"},
	} {
		if _, err := systemConfiguration(o); err == nil {
			t.Errorf("accepted invalid options: %+v", o)
		}
	}
	script, err := systemConfiguration(Options{})
	if err != nil || script != "" {
		t.Fatalf("empty options: %q, %v", script, err)
	}
}

func TestSystemConfigurationBeforePackages(t *testing.T) {
	steps, err := Steps("/tmp/L4T", Options{Username: "dev", Password: "test", Hostname: "jetson", Timezone: "Asia/Shanghai", Locale: "zh_CN.UTF-8", AptMirror: "https://mirror.example/ubuntu-ports", Packages: "curl", Wallpaper: "image.png"})
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 4 || steps[1].Name != "Configure chroot system" || steps[2].Name != "Install packages" || steps[3].Name != "Configure chroot appearance" {
		t.Fatalf("unexpected steps: %+v", steps)
	}
	script := steps[1].Command.Input
	if strings.Index(script, "sed -Ei") > strings.Index(script, "apt-get update") {
		t.Fatal("mirror must precede locale installation")
	}
	if !strings.Contains(steps[3].Command.Input, "dconf update") {
		t.Fatal("wallpaper database not compiled")
	}
}

func TestMirrorPreservesNvidiaSources(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash required")
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "sources.list.d"), 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"sources.list":                  "deb http://ports.ubuntu.com/ubuntu-ports/ jammy main\ndeb https://repo.download.nvidia.com/jetson/common r36.4 main\n",
		"sources.list.d/ubuntu.sources": "Types: deb\nURIs: http://ports.ubuntu.com/ubuntu-ports\nSuites: noble\nComponents: main\n",
		"sources.list.d/nvidia.list":    "deb https://repo.download.nvidia.com/jetson/t234 r36.4 main\n",
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	script, err := systemConfiguration(Options{AptMirror: "https://mirror.example/ubuntu-ports"})
	if err != nil {
		t.Fatal(err)
	}
	script = strings.ReplaceAll(script, "/etc/apt", dir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-s")
	cmd.Stdin = strings.NewReader(script)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("mirror script: %v: %s", err, output)
	}
	for name, original := range files {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		want := strings.ReplaceAll(original, "http://ports.ubuntu.com/ubuntu-ports/", "https://mirror.example/ubuntu-ports")
		want = strings.ReplaceAll(want, "http://ports.ubuntu.com/ubuntu-ports", "https://mirror.example/ubuntu-ports")
		if string(data) != want {
			t.Errorf("%s: got %q, want %q", name, data, want)
		}
	}
}
