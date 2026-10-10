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
	command := runner.Command{Dir: dir, Args: args}
	release, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return err
	}
	if isArch(string(release)) {
		if _, err := exec.LookPath("systemctl"); err != nil {
			return fmt.Errorf("Arch initrd flash needs systemctl: %w", err)
		}
		if _, err := exec.LookPath("exportfs"); err != nil {
			return fmt.Errorf("Arch initrd flash needs nfs-utils (exportfs): %w", err)
		}
		directory, err := serviceAdapter()
		if err != nil {
			return err
		}
		defer os.RemoveAll(directory)
		command.Env = []string{"PATH=" + directory + string(os.PathListSeparator) + os.Getenv("PATH")}
		emit("Arch: mapping NVIDIA's nfs-kernel-server service calls to systemd nfs-server.service for this flash only")
	}
	return runner.Run(ctx, command, emit)
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
	service := `#!/bin/sh
if [ "$#" -eq 2 ] && [ "$1" = "nfs-kernel-server" ]; then
  case "$2" in
    start|stop|restart|status) exec systemctl "$2" nfs-server.service ;;
  esac
fi
printf 'Unsupported NVIDIA service call: %s %s\n' "$1" "$2" >&2
exit 1
`
	if err := os.WriteFile(filepath.Join(directory, "service"), []byte(service), 0700); err != nil {
		os.RemoveAll(directory)
		return "", err
	}
	return directory, nil
}
