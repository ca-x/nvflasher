package capability

import (
	"nvflasher/internal/tegra"
	"os"
	"os/exec"
	"runtime"
)

type Status struct {
	Host   bool          `json:"host"`
	Root   bool          `json:"root"`
	Checks []tegra.Check `json:"checks"`
}

func Detect() Status {
	s := Status{Host: runtime.GOOS == "linux" && runtime.GOARCH == "amd64", Root: os.Geteuid() == 0, Checks: []tegra.Check{}}
	if !s.Host {
		return s
	}
	for _, name := range []string{"qemu-aarch64-static", "dpkg", "abootimg", "sshpass", "xmllint", "dtc", "ssh-keygen", "pkexec"} {
		status, hint := "ok", "Ready"
		if _, err := exec.LookPath(name); err != nil {
			status, hint = "warn", "Install "+name+" using your distribution package manager"
		}
		s.Checks = append(s.Checks, tegra.Check{Name: name, Status: status, Hint: hint})
	}
	if _, err := os.Stat("/proc/sys/fs/binfmt_misc/qemu-aarch64"); err != nil {
		s.Checks = append(s.Checks, tegra.Check{Name: "aarch64 binfmt", Status: "warn", Hint: "Register qemu-aarch64 with binfmt_misc"})
	}
	if _, err := exec.LookPath("exportfs"); err != nil {
		s.Checks = append(s.Checks, tegra.Check{Name: "NFS server", Status: "warn", Hint: "Install nfs-utils (Arch) or nfs-kernel-server (Debian)"})
	}
	return s
}
