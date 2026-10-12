package provision

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func validateCustomization(o Options) error {
	for name, value := range map[string]string{"suspend": o.Suspend, "screenLock": o.ScreenLock, "screenBlank": o.ScreenBlank} {
		if value != "" && value != "enable" && value != "disable" {
			return fmt.Errorf("invalid %s: use blank, enable or disable", name)
		}
	}
	if o.Autostart != "" {
		if !filepath.IsAbs(o.Autostart) || filepath.Clean(o.Autostart) != o.Autostart || o.Autostart == "/" || strings.ContainsAny(o.Autostart, "\r\n\t%\\\"`$") {
			return fmt.Errorf("autostart must be a single absolute executable path without desktop field codes")
		}
	}
	for _, path := range debPaths(o.DebFiles) {
		if !filepath.IsAbs(path) || !strings.HasSuffix(path, ".deb") {
			return fmt.Errorf("local DEB must be an absolute host .deb path: %q", path)
		}
	}
	return nil
}
func debPaths(value string) []string {
	var paths []string
	for _, line := range strings.Split(value, "\n") {
		if path := strings.TrimSpace(line); path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}
func checkDeb(ctx context.Context, path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("DEB must be a regular file: %s", path)
	}
	output, err := exec.CommandContext(ctx, "dpkg-deb", "--field", path, "Architecture").Output()
	if err != nil {
		return fmt.Errorf("inspect DEB %s: %w", path, err)
	}
	arch := strings.TrimSpace(string(output))
	if arch != "arm64" && arch != "all" {
		return fmt.Errorf("DEB architecture must be arm64 or all, got %q: %s", arch, path)
	}
	return nil
}
func validateDebFiles(ctx context.Context, value string) error {
	for _, path := range debPaths(value) {
		if err := checkDeb(ctx, path); err != nil {
			return err
		}
	}
	return nil
}
func stageDebFiles(ctx context.Context, root, value string) (string, error) {
	if err := appearanceDirectory(root, "tmp"); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(filepath.Join(root, "tmp"), "nvflasher-debs-")
	if err != nil {
		return "", err
	}
	// APT's unprivileged download helper must be able to read local packages.
	if err = os.Chmod(stage, 0755); err != nil {
		return "", errors.Join(err, os.RemoveAll(stage))
	}
	fail := func(err error) (string, error) { return "", errors.Join(err, os.RemoveAll(stage)) }
	for i, path := range debPaths(value) {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		src, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
		if err != nil {
			return fail(err)
		}
		info, err := src.Stat()
		if err != nil || !info.Mode().IsRegular() {
			src.Close()
			return fail(fmt.Errorf("DEB source must be a regular file: %s", path))
		}
		dstPath := filepath.Join(stage, fmt.Sprintf("package-%d.deb", i))
		dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			src.Close()
			return fail(err)
		}
		_, copyErr := io.Copy(dst, src)
		if err := errors.Join(copyErr, src.Close(), dst.Close()); err != nil {
			return fail(err)
		}
		if err := checkDeb(ctx, dstPath); err != nil {
			return fail(err)
		}
	}
	return stage, nil
}
func desktopConfiguration(o Options) string {
	if o.Suspend == "" && o.ScreenLock == "" && o.ScreenBlank == "" && !o.ChineseSupport && o.Autostart == "" {
		return ""
	}
	script := "set -eu\n"
	if len(desktopSchemas(o)) > 0 {
		script += "dconf update\n"
	}
	if o.Autostart != "" {
		script += "chown '" + o.Username + "' '/home/" + o.Username + "/.config' '/home/" + o.Username + "/.config/autostart' '/home/" + o.Username + "/.config/autostart/nvflasher.desktop'\n"
	}
	return script
}

func configureDesktop(root string, o Options, run func(string) error) error {
	if err := run(desktopPreflight(o)); err != nil {
		return err
	}
	if err := installDesktop(root, o); err != nil {
		return err
	}
	return run(desktopConfiguration(o))
}

// Run after package installation, but before writing any desktop defaults.
func desktopPreflight(o Options) string {
	script := "set -eu\n"
	if o.Suspend != "" || o.ScreenLock != "" || o.ScreenBlank != "" || o.ChineseSupport {
		script += "command -v dconf >/dev/null && command -v gsettings >/dev/null || { echo 'Desktop defaults require GNOME, dconf-cli and gsettings'; exit 1; }\n"
		for _, schema := range desktopSchemas(o) {
			script += "gsettings list-schemas | grep -Fxq '" + schema + "' || { echo 'Required GNOME schema missing: " + schema + "'; exit 1; }\n"
		}
	}
	if o.Autostart != "" {
		script += "test \"$(id -u '" + o.Username + "')\" -ne 0\n"
	}
	return script
}
func desktopSchemas(o Options) []string {
	var schemas []string
	if o.Suspend != "" {
		schemas = append(schemas, "org.gnome.settings-daemon.plugins.power")
	}
	if o.ScreenLock != "" {
		schemas = append(schemas, "org.gnome.desktop.screensaver")
	}
	if o.ScreenBlank != "" {
		schemas = append(schemas, "org.gnome.desktop.session")
	}
	if o.ChineseSupport {
		schemas = append(schemas, "org.gnome.desktop.input-sources")
	}
	return schemas
}
func installDesktop(root string, o Options) error {
	var defaults strings.Builder
	writeDefaults := func(name string) error {
		err := appearanceFile(root, "etc/dconf/db/nvflasher.d/"+name, []byte(defaults.String()))
		defaults.Reset()
		return err
	}
	if o.Suspend != "" {
		mode := "'suspend'"
		if o.Suspend == "disable" {
			mode = "'nothing'"
		}
		fmt.Fprintf(&defaults, "[org/gnome/settings-daemon/plugins/power]\nsleep-inactive-ac-type=%s\nsleep-inactive-battery-type=%s\nsleep-inactive-ac-timeout=1800\nsleep-inactive-battery-timeout=1800\n\n", mode, mode)
		if err := writeDefaults("10-suspend"); err != nil {
			return err
		}
	}
	if o.ScreenLock != "" {
		fmt.Fprintf(&defaults, "[org/gnome/desktop/screensaver]\nlock-enabled=%t\n\n", o.ScreenLock == "enable")
		if err := writeDefaults("11-screen-lock"); err != nil {
			return err
		}
	}
	if o.ScreenBlank != "" {
		delay := 300
		if o.ScreenBlank == "disable" {
			delay = 0
		}
		fmt.Fprintf(&defaults, "[org/gnome/desktop/session]\nidle-delay=uint32 %d\n\n", delay)
		if err := writeDefaults("12-screen-blank"); err != nil {
			return err
		}
	}
	if o.ChineseSupport {
		defaults.WriteString("[org/gnome/desktop/input-sources]\nsources=[('xkb', 'us'), ('ibus', 'pinyin')]\n\n")
		if err := writeDefaults("13-input-sources"); err != nil {
			return err
		}
	}
	if o.Suspend != "" || o.ScreenLock != "" || o.ScreenBlank != "" || o.ChineseSupport {
		if err := appearanceDirectory(root, "etc/dconf/profile"); err != nil {
			return err
		}
		profile := filepath.Join(root, "etc/dconf/profile/user")
		if err := safeTarget(root, profile); err != nil {
			return err
		}
		data, err := os.ReadFile(profile)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if len(data) == 0 {
			data = []byte("user-db:user\n")
		}
		found := false
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "system-db:nvflasher" {
				found = true
			}
		}
		if !found {
			data = append(data, []byte("\nsystem-db:nvflasher\n")...)
			if err := os.WriteFile(profile, data, 0644); err != nil {
				return err
			}
		}
	}
	if o.Autostart != "" {
		target := filepath.Join(root, strings.TrimPrefix(o.Autostart, "/"))
		if err := safeTarget(root, target); err != nil {
			return err
		}
		info, err := os.Stat(target)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			return fmt.Errorf("autostart target must be an executable regular file in rootfs")
		}
		entry := "[Desktop Entry]\nType=Application\nName=nvflasher application\nExec=\"" + o.Autostart + "\"\nTerminal=false\nX-GNOME-Autostart-enabled=true\n"
		if err := appearanceFile(root, "home/"+o.Username+"/.config/autostart/nvflasher.desktop", []byte(entry)); err != nil {
			return err
		}
	}
	return nil
}
