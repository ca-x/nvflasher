package capability

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"nvflasher/internal/tegra"
)

type InstallPlan struct {
	Distribution   string     `json:"distribution"`
	Packages       []string   `json:"packages"`
	Commands       [][]string `json:"commands"`
	ManualCommands [][]string `json:"manualCommands"`
}

func DetectInstallPlan() (InstallPlan, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return InstallPlan{}, err
	}
	return PlanFor(string(data), Detect().Checks)
}

func PlanFor(osRelease string, checks []tegra.Check) (InstallPlan, error) {
	properties := map[string]string{}
	for line := range strings.SplitSeq(osRelease, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			properties[key] = strings.Trim(value, "\"'")
		}
	}
	distribution := properties["ID"]
	identifiers := strings.Fields(distribution + " " + properties["ID_LIKE"])
	family := ""
	for _, id := range identifiers {
		switch id {
		case "arch", "debian", "ubuntu":
			family = id
		}
		if family != "" {
			break
		}
	}
	if family == "" {
		return InstallPlan{}, fmt.Errorf("unsupported package manager for %q", distribution)
	}
	packagesByCheck := map[string]map[string]string{
		"arch":   {"qemu-aarch64-static": "qemu-user-static", "aarch64 binfmt": "qemu-user-static-binfmt", "dpkg": "dpkg", "sshpass": "sshpass", "xmllint": "libxml2", "dtc": "dtc", "ssh-keygen": "openssh", "pkexec": "polkit", "NFS server": "nfs-utils"},
		"debian": {"qemu-aarch64-static": "qemu-user-static", "aarch64 binfmt": "qemu-user-static-binfmt", "abootimg": "abootimg", "sshpass": "sshpass", "xmllint": "libxml2-utils", "dtc": "device-tree-compiler", "ssh-keygen": "openssh-client", "pkexec": "polkitd", "NFS server": "nfs-kernel-server"},
		"ubuntu": {"qemu-aarch64-static": "qemu-user-static", "aarch64 binfmt": "qemu-user-static-binfmt", "abootimg": "abootimg", "sshpass": "sshpass", "xmllint": "libxml2-utils", "dtc": "device-tree-compiler", "ssh-keygen": "openssh-client", "pkexec": "polkitd", "NFS server": "nfs-kernel-server"},
	}
	selected := map[string]bool{}
	for _, check := range checks {
		if check.Status == "warn" {
			if name := packagesByCheck[family][check.Name]; name != "" {
				selected[name] = true
			}
		}
	}
	plan := InstallPlan{Distribution: distribution, Packages: []string{}, Commands: [][]string{}, ManualCommands: [][]string{}}
	if family == "arch" {
		for _, check := range checks {
			if check.Name == "abootimg" && check.Status == "warn" {
				plan.ManualCommands = append(plan.ManualCommands, []string{"paru", "-S", "abootimg"})
			}
		}
	}
	for name := range selected {
		plan.Packages = append(plan.Packages, name)
	}
	slices.Sort(plan.Packages)
	if len(plan.Packages) == 0 {
		return plan, nil
	}
	if family == "arch" {
		plan.Commands = append(plan.Commands, append([]string{"pacman", "-Syu", "--needed", "--noconfirm"}, plan.Packages...))
	} else {
		plan.Commands = append(plan.Commands, []string{"apt-get", "update"}, append([]string{"apt-get", "install", "-y"}, plan.Packages...))
	}
	return plan, nil
}
