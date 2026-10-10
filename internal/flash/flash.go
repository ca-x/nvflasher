package flash

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nvflasher/internal/runner"
	"nvflasher/internal/tegra"
)

func Run(ctx context.Context, dir string, opts tegra.Options, emit func(string)) error {
	args, err := tegra.Command(opts)
	if err != nil {
		return err
	}
	if info, err := os.Stat(filepath.Join(dir, opts.Board+".conf")); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("selected board configuration is unavailable in this BSP: %s.conf", opts.Board)
	}
	if err := cleanupStaleImage(ctx, dir, emit); err != nil {
		return err
	}
	if err := checkImageLoops(ctx, dir); err != nil {
		return err
	}
	command := runner.Command{Dir: dir, Args: args}
	release, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return err
	}
	arch := isArch(string(release))
	_, ping6Err := exec.LookPath("ping6")
	var ping string
	if ping6Err != nil {
		ping, err = exec.LookPath("ping")
		if err != nil {
			return fmt.Errorf("initrd flash needs ping6 or ping with IPv6 support: %w", err)
		}
	}
	if arch || ping6Err != nil {
		directory, err := os.MkdirTemp("", "nvflasher-compat-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(directory)
		command.Env = []string{"PATH=" + directory + string(os.PathListSeparator) + os.Getenv("PATH")}
		if ping6Err != nil {
			if err := writePing6Adapter(directory, ping); err != nil {
				return err
			}
			emit("Compatibility: ping6 is unavailable; using ping -6 for this flash only")
		}
		if arch {
			if err := writeServiceAdapter(directory); err != nil {
				return err
			}
		}
	}
	if arch {
		if _, err := exec.LookPath("systemctl"); err != nil {
			return fmt.Errorf("Arch initrd flash needs systemctl: %w", err)
		}
		if _, err := exec.LookPath("exportfs"); err != nil {
			return fmt.Errorf("Arch initrd flash needs nfs-utils (exportfs): %w", err)
		}
		emit("Arch: mapping NVIDIA's nfs-kernel-server service calls to systemd nfs-server.service for this flash only")
	}
	return runChecked(ctx, command, emit)
}

func isArch(release string) bool {
	for line := range strings.SplitSeq(release, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || key != "ID" && key != "ID_LIKE" {
			continue
		}
		for _, identifier := range strings.Fields(strings.Trim(value, "\"'")) {
			if identifier == "arch" {
				return true
			}
		}
	}
	return false
}

func serviceAdapter() (string, error) {
	directory, err := os.MkdirTemp("", "nvflasher-service-")
	if err != nil {
		return "", err
	}
	if err := writeServiceAdapter(directory); err != nil {
		os.RemoveAll(directory)
		return "", err
	}
	return directory, nil
}

func writePing6Adapter(directory, ping string) error {
	// Quote the resolved executable path; forward every NVIDIA argument unchanged.
	quoted := "'" + strings.ReplaceAll(ping, "'", "'\"'\"'") + "'"
	return os.WriteFile(filepath.Join(directory, "ping6"), []byte("#!/bin/sh\nexec "+quoted+" -6 \"$@\"\n"), 0700)
}

func writeServiceAdapter(directory string) error {
	service := `#!/bin/sh
if [ "$#" -eq 2 ] && [ "$1" = "nfs-kernel-server" ]; then
  case "$2" in
    start|stop|restart|status) exec systemctl "$2" nfs-server.service ;;
  esac
fi
printf 'Unsupported NVIDIA service call: %s %s\n' "$1" "$2" >&2
exit 1
`
	return os.WriteFile(filepath.Join(directory, "service"), []byte(service), 0700)
}
