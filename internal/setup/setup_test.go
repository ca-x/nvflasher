package setup

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestCatalog(t *testing.T) {
	super, err := Lookup("orin-nano-super-devkit")
	if err != nil || super.Board != "jetson-orin-nano-devkit-super" {
		t.Fatalf("unexpected Super config: %+v, %v", super, err)
	}
	model, err := Lookup("orin-nano-8gb-devkit")
	if err != nil || model.Release != "36.5.0" || len(model.Packages) != 2 {
		t.Fatalf("unexpected model: %+v, %v", model, err)
	}
	if model.Board != "jetson-orin-nano-devkit" || model.Packages[0] != super.Packages[0] {
		t.Fatal("Super and 8GB devkit should share BSP, not flash board config")
	}
	for _, pkg := range model.Packages {
		if !strings.HasPrefix(pkg.URL, "https://developer.nvidia.com/") || len(pkg.SHA1) != 40 {
			t.Errorf("invalid official resource: %+v", pkg)
		}
	}
	if _, err := Lookup("unknown"); err == nil {
		t.Fatal("unsupported model accepted")
	}
}

func TestExtractReportsProgress(t *testing.T) {
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("tar unavailable")
	}
	input := t.TempDir()
	if err := os.WriteFile(filepath.Join(input, "payload"), []byte(strings.Repeat("data", 8192)), 0600); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "sample.tbz2")
	if output, err := exec.CommandContext(t.Context(), "tar", "-cjf", archive, "-C", input, "payload").CombinedOutput(); err != nil {
		t.Fatalf("build archive: %v: %s", err, output)
	}
	target := t.TempDir()
	var progress []string
	if err := extract(t.Context(), archive, target, "rootfs", func(line string) { progress = append(progress, line) }); err != nil {
		t.Fatal(err)
	}
	if len(progress) == 0 || progress[len(progress)-1] != "rootfs: 100%" {
		t.Fatalf("missing extraction progress: %v", progress)
	}
	if data, err := os.ReadFile(filepath.Join(target, "payload")); err != nil || len(data) != 4*8192 {
		t.Fatalf("extracted contents: %d bytes, %v", len(data), err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := extract(ctx, archive, t.TempDir(), "rootfs", func(string) {}); err == nil {
		t.Fatal("cancelled extraction succeeded")
	}
}

func TestPrepareResumesAfterApplyFailure(t *testing.T) {
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("tar unavailable")
	}
	directory := t.TempDir()
	source := t.TempDir()
	l4t := filepath.Join(source, "Linux_for_Tegra")
	for _, name := range []string{"tools/kernel_flash/l4t_initrd_flash.sh", "apply_binaries.sh"} {
		path := filepath.Join(l4t, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		contents := "tool"
		if name == "apply_binaries.sh" {
			contents = "#!/bin/sh\n[ -f rootfs/etc/os-release ] || exit 4\n[ -n \"$PATH\" ] || exit 5\nif [ ! -f .retried ]; then touch .retried; exit 2; fi\nprintf 'R36 (release), REVISION: 5.0' > rootfs/etc/nv_tegra_release\n"
		}
		if err := os.WriteFile(path, []byte(contents), 0755); err != nil {
			t.Fatal(err)
		}
	}
	rootfs := filepath.Join(t.TempDir(), "etc")
	if err := os.MkdirAll(rootfs, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootfs, "os-release"), []byte("ID=ubuntu\n"), 0644); err != nil {
		t.Fatal(err)
	}
	packages := make([]Package, 2)
	for index, item := range []struct{ input, entry string }{{source, "Linux_for_Tegra"}, {filepath.Dir(rootfs), "etc"}} {
		archive := filepath.Join(directory, []string{"bsp.tbz2", "rootfs.tbz2"}[index])
		if output, err := exec.CommandContext(t.Context(), "tar", "-cjf", archive, "-C", item.input, item.entry).CombinedOutput(); err != nil {
			t.Fatalf("make archive: %v: %s", err, output)
		}
		data, err := os.ReadFile(archive)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha1.Sum(data)
		packages[index] = Package{Name: filepath.Base(archive), SHA1: hex.EncodeToString(hash[:])}
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "dpkg"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	model := Model{ID: "orin-nano-8gb-devkit", Release: "36.5.0", Packages: packages}
	t.Setenv("PATH", t.TempDir())
	if _, err := prepare(t.Context(), model, directory, func(string) {}); err == nil || !strings.Contains(err.Error(), "dpkg") {
		t.Fatalf("missing host package not detected: %v", err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+"/usr/bin:/bin")
	if _, err := prepare(t.Context(), model, directory, func(string) {}); err == nil {
		t.Fatal("apply failure should be reported")
	}
	var events []string
	prepared, err := prepare(t.Context(), model, directory, func(line string) { events = append(events, line) })
	if err != nil || !strings.HasSuffix(prepared, "Linux_for_Tegra") {
		t.Fatalf("retry: %s, %v", prepared, err)
	}
	if !strings.Contains(strings.Join(events, "\n"), "Reusing extracted sample rootfs") {
		t.Fatalf("retry re-extracted instead of resuming: %v", events)
	}
	if !fileExists(filepath.Join(prepared, "rootfs/etc/nv_tegra_release")) {
		t.Fatal("prepared rootfs missing")
	}
}

func TestVerifyArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.tbz2")
	if err := os.WriteFile(path, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verify(path, "a9993e364706816aba3e25717850c26c9cd0d89d"); err != nil {
		t.Fatal(err)
	}
	if err := verify(path, strings.Repeat("0", 40)); err == nil {
		t.Fatal("corrupt archive accepted")
	}
}

func TestRemoveStaleDeviceNodesRefusesUnexpectedFiles(t *testing.T) {
	rootfs := t.TempDir()
	dev := filepath.Join(rootfs, "dev")
	if err := os.Mkdir(dev, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dev, "random")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := removeStaleDeviceNodes(rootfs, func(string) {}); err == nil {
		t.Fatal("unexpected regular file was silently removed")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("unexpected file removed: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/dev/random", path); err != nil {
		t.Fatal(err)
	}
	if err := removeStaleDeviceNodes(rootfs, func(string) {}); err == nil {
		t.Fatal("unexpected symlink was silently removed")
	}
	if err := os.RemoveAll(dev); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/dev", dev); err != nil {
		t.Fatal(err)
	}
	if err := removeStaleDeviceNodes(rootfs, func(string) {}); err == nil {
		t.Fatal("symlinked device directory was silently accepted")
	}
}

func TestRemoveStaleDeviceNodesAllowsRetry(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("creating device nodes requires root")
	}
	rootfs := t.TempDir()
	dev := filepath.Join(rootfs, "dev")
	if err := os.Mkdir(dev, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dev, "random")
	host, err := os.Stat("/dev/random")
	if err != nil {
		t.Fatal(err)
	}
	device := host.Sys().(*syscall.Stat_t)
	if err := syscall.Mknod(path, syscall.S_IFCHR|0666, int(device.Rdev)); err != nil {
		t.Skipf("creating device nodes unavailable: %v", err)
	}
	var events []string
	if err := removeStaleDeviceNodes(rootfs, func(line string) { events = append(events, line) }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("stale node still present: %v", err)
	}
	if len(events) != 1 || !strings.Contains(events[0], "random") {
		t.Fatalf("missing removal feedback: %v", events)
	}
}
