package capability

import (
	"slices"
	"testing"

	"nvflasher/internal/tegra"
)

func TestPlanFor(t *testing.T) {
	checks := []tegra.Check{{Name: "dpkg", Status: "warn"}, {Name: "abootimg", Status: "warn"}, {Name: "sshpass", Status: "warn"}, {Name: "dtc", Status: "warn"}, {Name: "NFS server", Status: "warn"}, {Name: "xmllint", Status: "ok"}}
	for _, test := range []struct {
		name, osRelease string
		packages        []string
		command         string
	}{
		{"arch derivative", "ID=omarchy\nID_LIKE=arch\n", []string{"dpkg", "dtc", "nfs-utils", "sshpass"}, "pacman"},
		{"debian", "ID=debian\n", []string{"abootimg", "device-tree-compiler", "nfs-kernel-server", "sshpass"}, "apt-get"},
		{"ubuntu", "ID=ubuntu\nID_LIKE=debian\n", []string{"abootimg", "device-tree-compiler", "nfs-kernel-server", "sshpass"}, "apt-get"},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan, err := PlanFor(test.osRelease, checks)
			if err != nil || !slices.Equal(plan.Packages, test.packages) || len(plan.Commands) == 0 || plan.Commands[0][0] != test.command {
				t.Fatalf("plan=%+v err=%v", plan, err)
			}
			if test.name == "arch derivative" && (len(plan.ManualCommands) != 1 || !slices.Equal(plan.ManualCommands[0], []string{"paru", "-S", "abootimg"})) {
				t.Fatalf("missing manual AUR instruction: %+v", plan)
			}
		})
	}
	if _, err := PlanFor("ID=fedora\n", checks); err == nil {
		t.Fatal("unsupported distribution accepted")
	}
}

func TestPlanForOnlyAURDependency(t *testing.T) {
	plan, err := PlanFor("ID=omarchy\nID_LIKE=arch\n", []tegra.Check{{Name: "abootimg", Status: "warn"}})
	if err != nil || len(plan.Packages) != 0 || len(plan.Commands) != 0 || len(plan.ManualCommands) != 1 {
		t.Fatalf("AUR package must not be passed to pacman: %+v, %v", plan, err)
	}
}
