package provision

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// replaceChrootDNS moves the original entry without following rootfs symlinks.
// Renaming preserves dangling/absolute symlinks, ownership and permissions.
func replaceChrootDNS(root string, host []byte) (func() error, error) {
	target := filepath.Join(root, "etc/resolv.conf")
	if err := safeParent(root, target); err != nil {
		return nil, err
	}
	_, err := os.Lstat(target)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	backup := ""
	if exists {
		info, err := os.Lstat(target)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
			return nil, fmt.Errorf("unsupported DNS configuration entry: %s", target)
		}
		folder, err := os.MkdirTemp(filepath.Dir(target), ".nvflasher-dns-")
		if err != nil {
			return nil, err
		}
		backup = filepath.Join(folder, "resolv.conf")
		if err := os.Rename(target, backup); err != nil {
			return nil, errors.Join(err, os.Remove(folder))
		}
	}
	restore := func() error {
		if err := safeParent(root, target); err != nil {
			return err
		}
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove temporary DNS configuration: %w", err)
		}
		if backup != "" {
			if err := os.Rename(backup, target); err != nil {
				return fmt.Errorf("restore DNS configuration from %s: %w", backup, err)
			}
			return os.Remove(filepath.Dir(backup))
		}
		return nil
	}
	if err := os.WriteFile(target, host, 0644); err != nil {
		return nil, errors.Join(err, restore())
	}
	return restore, nil
}
