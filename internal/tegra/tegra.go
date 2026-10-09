package tegra

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Options struct {
	Board    string `json:"board"`
	Device   string `json:"device"`
	NVMe     string `json:"nvme"`
	QSPI     string `json:"qspi"`
	Erase    bool   `json:"erase"`
	ShowLogs bool   `json:"showLogs"`
}
type Preset struct {
	Name    string  `json:"name"`
	Options Options `json:"options"`
}

var Presets = []Preset{
	{"Orin Nano Super · NVMe", Options{"jetson-orin-nano-devkit-super", "nvme0n1p1", "tools/kernel_flash/flash_l4t_t234_nvme.xml", "bootloader/generic/cfg/flash_t234_qspi.xml", false, true}},
	{"Orin Nano · NVMe", Options{"jetson-orin-nano-devkit", "nvme0n1p1", "tools/kernel_flash/flash_l4t_t234_nvme.xml", "bootloader/generic/cfg/flash_t234_qspi.xml", false, true}},
	{"Orin NX · NVMe", Options{"jetson-orin-nano-devkit", "nvme0n1p1", "tools/kernel_flash/flash_l4t_t234_nvme.xml", "bootloader/generic/cfg/flash_t234_qspi.xml", false, true}},
}
var token = regexp.MustCompile(`^[a-zA-Z0-9_./+-]+$`)

func Command(o Options) ([]string, error) {
	for _, v := range []string{o.Board, o.Device, o.NVMe, o.QSPI} {
		if !token.MatchString(v) || strings.Contains(v, "..") {
			return nil, fmt.Errorf("invalid flashing parameter: %q", v)
		}
	}
	args := []string{"./tools/kernel_flash/l4t_initrd_flash.sh", "--external-device", o.Device, "-c", o.NVMe, "-p", "-c " + o.QSPI}
	if o.Erase {
		args = append(args, "--erase-all")
	}
	if o.ShowLogs {
		args = append(args, "--showlogs")
	}
	return append(args, "--network", "usb0", o.Board, "internal"), nil
}

type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Hint   string `json:"hint"`
}
type Report struct {
	Checks  []Check  `json:"checks"`
	Version string   `json:"version"`
	Boards  []string `json:"boards"`
}

func Inspect(dir string) Report {
	result := Report{Boards: []string{}, Checks: []Check{}}
	for _, item := range []struct {
		name, path string
		directory  bool
	}{{"flash tool", "tools/kernel_flash/l4t_initrd_flash.sh", false}, {"rootfs", "rootfs", true}, {"apply_binaries", "rootfs/etc/nv_tegra_release", false}} {
		status, hint := "ok", "Ready"
		info, err := os.Stat(filepath.Join(dir, item.path))
		if err != nil || info.IsDir() != item.directory {
			status, hint = "error", "Missing "+item.path+"; extract BSP/rootfs and run apply_binaries.sh"
		}
		result.Checks = append(result.Checks, Check{item.name, status, hint})
	}
	release, _ := os.ReadFile(filepath.Join(dir, "rootfs/etc/nv_tegra_release"))
	if m := regexp.MustCompile(`R\d+.*?REVISION:\s*[\d.]+`).Find(release); m != nil {
		result.Version = string(m)
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "*.conf"))
	for _, p := range paths {
		name := strings.TrimSuffix(filepath.Base(p), ".conf")
		if !strings.Contains(name, "common") {
			result.Boards = append(result.Boards, name)
		}
	}
	sort.Strings(result.Boards)
	return result
}
