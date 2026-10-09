package massflash

import (
	"context"
	"fmt"
	"nvflasher/internal/runner"
	"nvflasher/internal/tegra"
	"os"
	"path/filepath"
)

func Generate(ctx context.Context, dir string, opts tegra.Options, count int, emit func(string)) error {
	if count < 1 || count > 64 {
		return fmt.Errorf("invalid device limit")
	}
	args, err := tegra.Command(opts)
	if err != nil {
		return err
	}
	args = append(args[:len(args)-2], append([]string{"--no-flash", "--massflash", fmt.Sprint(count)}, args[len(args)-2:]...)...)
	return runner.Run(ctx, runner.Command{Dir: dir, Args: args}, emit)
}
func Flash(ctx context.Context, dir string, count int, show bool, emit func(string)) error {
	if count < 1 || count > 64 {
		return fmt.Errorf("invalid device limit")
	}
	if _, err := os.Stat(filepath.Join(dir, "tools/kernel_flash/l4t_initrd_flash.sh")); err != nil {
		return fmt.Errorf("invalid mfi directory: %w", err)
	}
	args := []string{"./tools/kernel_flash/l4t_initrd_flash.sh", "--flash-only", "--network", "usb0", "--massflash", fmt.Sprint(count)}
	if show {
		args = append(args, "--showlogs")
	}
	return runner.Run(ctx, runner.Command{Dir: dir, Args: args}, emit)
}
