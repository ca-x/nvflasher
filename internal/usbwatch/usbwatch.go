package usbwatch

import (
	"os"
	"path/filepath"
	"strings"
)

type Device struct {
	Path    string `json:"path"`
	Product string `json:"product"`
	Mode    string `json:"mode"`
}

func Parse(vendor, product string) string {
	if strings.ToLower(strings.TrimSpace(vendor)) != "0955" {
		return ""
	}
	p := strings.ToLower(strings.TrimSpace(product))
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
			out = append(out, Device{dir.Name(), strings.TrimSpace(string(p)), mode})
		}
	}
	return out
}
