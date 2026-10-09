package capability

import (
	"slices"
	"testing"

	"nvflasher/internal/tegra"
)

func TestPlanFor(t *testing.T) {
	checks := []tegra.Check{{Name: "dpkg", Status: "warn"}, {Name: "sshpass", Status: "warn"}, {Name: "dtc", Status: "warn"}, {Name: "NFS server", Status: "warn"}, {Name: "xmllint", Status: "ok"}}
	for _, test := range []struct {
		name, osRelease string
		packages        []string
		command         string
	}{
		{"arch derivative", "ID=omarchy\nID_LIKE=arch\n", []string{"dpkg", "dtc", "nfs-utils", "sshpass"}, "pacman"},
		{"debian", "ID=debian\n", []string{"device-tree-compiler", "nfs-kernel-server", "sshpass"}, "apt-get"},
		{"ubuntu", "ID=ubuntu\nID_LIKE=debian\n", []string{"device-tree-compiler", "nfs-kernel-server", "sshpass"}, "apt-get"},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan, err := PlanFor(test.osRelease, checks)
			if err != nil || !slices.Equal(plan.Packages, test.packages) || len(plan.Commands) == 0 || plan.Commands[0][0] != test.command {
				t.Fatalf("plan=%+v err=%v", plan, err)
			}
		})
	}
	if _, err := PlanFor("ID=fedora\n", checks); err == nil {
		t.Fatal("unsupported distribution accepted")
	}
}
