package flash

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
)

// CleanupImages checks and cleans only idle standard BSP image resources.
func CleanupImages(ctx context.Context, dir string, emit func(string)) error {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return fmt.Errorf("resolve BSP directory: %w", err)
	}
	emit("Checking image resources: " + absolute)
	if err := cleanupStaleImage(ctx, absolute, emit); err != nil {
		return err
	}
	if err := checkImageLoops(ctx, absolute); err != nil {
		return err
	}
	emit("Image resource check complete: no attached BSP image remains; flashing was not started.")
	return nil
}

// Only the standard interrupted image-generation mount is eligible. Never
// force unmount, remove images, or detach a loop whose identity has changed.
func cleanupStaleImage(ctx context.Context, dir string, emit func(string)) error {
	image := filepath.Join(dir, "bootloader", "system.img")
	mount := filepath.Join(dir, "bootloader", "mnt")
	output, err := exec.CommandContext(ctx, "losetup", "--json", "--list", "--output", "NAME,BACK-FILE").Output()
	if err != nil {
		return fmt.Errorf("verify stale image: %w", err)
	}
	var loops struct {
		Devices []struct {
			Name string `json:"name"`
			File string `json:"back-file"`
		} `json:"loopdevices"`
	}
	if err := json.Unmarshal(output, &loops); err != nil {
		return err
	}
	for _, loop := range loops.Devices {
		if loop.File != image {
			continue
		}
		// Do not interfere with another NVIDIA run, even between file accesses.
		check := exec.CommandContext(ctx, "pgrep", "-f", "[l]4t_initrd_flash|[t]egraflash|[m]ksparse|[m]kfs|[e]2fsck")
		if err := check.Run(); err == nil {
			return fmt.Errorf("refusing cleanup: image-generation or flash processes are active")
		} else if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 || ctx.Err() != nil {
			return fmt.Errorf("cannot verify active image processes: %w", err)
		}
		// fuser must explicitly report no users. Any other exit is unsafe.
		idle := func(args ...string) bool {
			output, err := exec.CommandContext(ctx, "fuser", args...).CombinedOutput()
			if len(output) > 0 {
				emit("Image resource users: " + string(output))
			}
			exit, ok := err.(*exec.ExitError)
			return ok && exit.ExitCode() == 1 && ctx.Err() == nil
		}
		mounted, err := exec.CommandContext(ctx, "findmnt", "-rn", "-S", loop.Name, "-o", "TARGET").Output()
		if err == nil {
			if string(mounted) != mount+"\n" {
				return fmt.Errorf("refusing cleanup: %s has unexpected mounts", loop.Name)
			}
			if !idle("-m", mount) {
				return fmt.Errorf("refusing cleanup: image mount is busy or its users cannot be verified")
			}
			if !idle(image, loop.Name) {
				return fmt.Errorf("refusing cleanup: image or loop device is busy")
			}
			// Recheck the identity immediately before changing mount state.
			if err := verifyLoop(ctx, loop.Name, image); err != nil {
				return err
			}
			emit("Cleaning idle interrupted image mount: " + mount)
			if out, err := exec.CommandContext(ctx, "umount", "--", mount).CombinedOutput(); err != nil {
				return fmt.Errorf("safe image unmount failed: %w: %s", err, out)
			}
		} else {
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 {
				return fmt.Errorf("cannot verify image mounts: %w", err)
			}
		}
		if !idle(image, loop.Name) {
			return fmt.Errorf("refusing cleanup: image or loop device is busy")
		}
		if err := verifyLoop(ctx, loop.Name, image); err != nil {
			return err
		}
		if out, err := exec.CommandContext(ctx, "losetup", "--detach", loop.Name).CombinedOutput(); err != nil {
			return fmt.Errorf("safe image detach failed: %w: %s", err, out)
		}
		emit("Detached idle image loop: " + loop.Name)
	}
	return nil
}

func verifyLoop(ctx context.Context, device, image string) error {
	out, err := exec.CommandContext(ctx, "losetup", "--noheadings", "--output", "BACK-FILE", device).Output()
	if err != nil || string(out) != image+"\n" {
		return fmt.Errorf("loop backing file changed or cannot be verified: %s", device)
	}
	return nil
}
