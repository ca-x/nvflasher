package flash

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Never recreate a BSP image while it is still attached to a loop device.
// Truncating an attached backing file can invalidate cached filesystem data.
func checkImageLoops(ctx context.Context, dir string) error {
	output, err := exec.CommandContext(ctx, "losetup", "--list", "--noheadings", "--output", "BACK-FILE").Output()
	if err != nil {
		return fmt.Errorf("cannot verify BSP image loop devices: %w", err)
	}
	return rejectImageLoops(dir, string(output))
}

func rejectImageLoops(dir, output string) error {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(absolute); err == nil {
		absolute = resolved
	}
	for _, line := range strings.Split(output, "\n") {
		image := strings.TrimSpace(line)
		if image == "" {
			continue
		}
		relative, err := filepath.Rel(absolute, image)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
			return fmt.Errorf("BSP image still attached to a loop device: %s; stop image users and safely unmount/detach it before generating images (not done automatically)", image)
		}
	}
	return nil
}
