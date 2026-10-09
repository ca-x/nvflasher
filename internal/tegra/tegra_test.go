package tegra

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCommand(t *testing.T) {
	args, err := Command(Presets[0].Options)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"./tools/kernel_flash/l4t_initrd_flash.sh", "--external-device", "nvme0n1p1", "-c", "tools/kernel_flash/flash_l4t_t234_nvme.xml", "-p", "-c bootloader/generic/cfg/flash_t234_qspi.xml", "--showlogs", "--network", "usb0", "jetson-orin-nano-devkit-super", "internal"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("%v", args)
	}
	bad := Presets[0].Options
	bad.Board = "foo; rm -rf /"
	if _, err = Command(bad); err == nil {
		t.Fatal("expected invalid board")
	}
}
func TestInspect(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{"tools/kernel_flash/l4t_initrd_flash.sh", "rootfs/etc/nv_tegra_release", "jetson-orin-nano-devkit-super.conf", "common.conf"} {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("# R36 (release), REVISION: 4.3"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	report := Inspect(dir)
	if report.Version == "" || len(report.Boards) != 1 || report.Checks[2].Status != "ok" {
		t.Fatalf("%+v", report)
	}
}
func TestInspectIncompleteApply(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "rootfs"), 0755); err != nil {
		t.Fatal(err)
	}
	report := Inspect(dir)
	if report.Checks[2].Status != "error" || !strings.Contains(report.Checks[2].Hint, "Download step") {
		t.Fatalf("unhelpful preparation hint: %+v", report.Checks[2])
	}
}
