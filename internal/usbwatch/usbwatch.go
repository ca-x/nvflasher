package usbwatch

import (
	"os"
	"path/filepath"
	"strings"
)

type Device struct {
	Path     string `json:"path"`
	Product  string `json:"product"`
	Mode     string `json:"mode"`
	BoardIDs string `json:"boardIds"`
	Release  string `json:"release"`
}

func Parse(vendor, product string) string {
	if strings.ToLower(strings.TrimSpace(vendor)) != "0955" {
		return ""
	}
	p := strings.ToLower(strings.TrimSpace(product))
	if p == "7020" {
		return "running"
	}
	if p == "7035" {
		return "initrd"
	}
	if len(p) == 4 && p[0] == '7' && p[2:] == "23" && strings.ContainsRune("0123456789abcdef", rune(p[1])) {
		return "recovery"
	}
	return ""
}
func Scan(root string) []Device {
	dirs, _ := os.ReadDir(root)
	out := []Device{}
	for _, dir := range dirs {
		base := filepath.Join(root, dir.Name())
		v, e1 := os.ReadFile(filepath.Join(base, "idVendor"))
		p, e2 := os.ReadFile(filepath.Join(base, "idProduct"))
		if e1 != nil || e2 != nil {
			continue
		}
		mode := Parse(string(v), string(p))
		if mode != "" {
			out = append(out, Device{Path: dir.Name(), Product: strings.TrimSpace(string(p)), Mode: mode})
		}
	}
	running := -1
	for index, device := range out {
		if device.Mode == "running" {
			if running >= 0 {
				return out
			}
			running = index
		}
	}
	if running >= 0 && root == "/sys/bus/usb/devices" {
		for _, pattern := range []string{"/run/media/*/L4T-README/version/chosen/ids", "/media/*/L4T-README/version/chosen/ids"} {
			paths, _ := filepath.Glob(pattern)
			if len(paths) != 1 {
				continue
			}
			ids, err := os.ReadFile(paths[0])
			if err == nil {
				out[running].BoardIDs = BoardIDsFromReadme(ids)
				packageInfo, readErr := os.ReadFile(filepath.Join(filepath.Dir(filepath.Dir(paths[0])), "nvidia-l4t-core.dpkg-s.txt"))
				if readErr == nil {
					out[running].Release = ReleaseFromPackage(packageInfo)
				}
				break
			}
		}
	}
	return out
}

func ReleaseFromPackage(packageInfo []byte) string {
	for line := range strings.SplitSeq(string(packageInfo), "\n") {
		if version, ok := strings.CutPrefix(line, "Version: "); ok {
			release, _, _ := strings.Cut(strings.TrimSpace(version), "-")
			parts := strings.Split(release, ".")
			if len(parts) == 3 {
				valid := true
				for _, part := range parts {
					if part == "" {
						valid = false
					}
					for _, char := range part {
						if char < '0' || char > '9' {
							valid = false
						}
					}
				}
				if valid {
					return release
				}
			}
		}
	}
	return ""
}

func BoardIDsFromReadme(ids []byte) string {
	fields := strings.Fields(strings.ReplaceAll(string(ids), "\x00", " "))
	result := []string{}
	for _, field := range fields {
		if len(field) != 13 || field[4] != '-' || field[9] != '-' {
			continue
		}
		valid := true
		for index, char := range field {
			if index != 4 && index != 9 && (char < '0' || char > '9') {
				valid = false
				break
			}
		}
		if valid {
			result = append(result, field)
		}
	}
	return strings.Join(result, " / ")
}
