package provision

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"nvflasher/internal/runner"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Options struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	Hostname       string `json:"hostname"`
	Autologin      bool   `json:"autologin"`
	PublicKey      string `json:"publicKey"`
	Packages       string `json:"packages"`
	Script         string `json:"script"`
	Overlay        string `json:"overlay"`
	ClearKeys      bool   `json:"clearKeys"`
	ClearMachineID bool   `json:"clearMachineId"`
	FirstBoot      bool   `json:"firstBoot"`
}
type Step struct {
	Name    string
	Command runner.Command
}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
var packageName = regexp.MustCompile(`^[a-zA-Z0-9.+:-]+$`)

func GeneratePassword() (string, error) {
	const alphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	for i := range raw {
		raw[i] = alphabet[int(raw[i])%len(alphabet)]
	}
	return string(raw), nil
}
func Steps(dir string, o Options) ([]Step, error) {
	if !identifier.MatchString(o.Username) || !identifier.MatchString(o.Hostname) || o.Password == "" {
		return nil, fmt.Errorf("invalid user, hostname or empty password")
	}
	root := filepath.Join(dir, "rootfs")
	args := []string{"./tools/l4t_create_default_user.sh", "-u", o.Username, "-p", o.Password, "-n", o.Hostname, "--accept-license"}
	if o.Autologin {
		args = append(args, "-a")
	}
	steps := []Step{{"Create default user", runner.Command{Dir: dir, Args: args}}}
	if o.Overlay != "" {
		steps = append(steps, Step{"Apply overlay", runner.Command{Args: []string{"cp", "-a", filepath.Join(o.Overlay, "."), root}}})
	}
	packages := strings.Fields(o.Packages)
	for _, p := range packages {
		if !packageName.MatchString(p) {
			return nil, fmt.Errorf("invalid package: %q", p)
		}
	}
	if len(packages) > 0 {
		script := "export DEBIAN_FRONTEND=noninteractive\napt-get update && apt-get install -y --no-install-recommends " + strings.Join(packages, " ") + "\n"
		steps = append(steps, Step{"Install packages", runner.Command{Args: []string{"chroot", root, "/bin/bash", "-s"}, Input: script}})
	}
	if strings.TrimSpace(o.Script) != "" {
		steps = append(steps, Step{"Run chroot script", runner.Command{Args: []string{"chroot", root, "/bin/bash", "-s"}, Input: "export DEBIAN_FRONTEND=noninteractive\nset -e\n" + o.Script}})
	}
	return steps, nil
}
func Run(ctx context.Context, dir string, o Options, emit func(string)) error {
	steps, err := Steps(dir, o)
	if err != nil {
		return err
	}
	root := filepath.Join(dir, "rootfs")
	if _, err = os.Stat(filepath.Join(root, "etc/nv_tegra_release")); err != nil {
		return fmt.Errorf("apply_binaries is required: %w", err)
	}
	if o.Overlay != "" {
		if info, err := os.Stat(o.Overlay); err != nil || !info.IsDir() {
			return fmt.Errorf("invalid overlay directory")
		}
	}
	for _, step := range steps {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		emit("Step: " + step.Name)
		if strings.HasPrefix(step.Name, "Install") || step.Name == "Run chroot script" {
			err = withChroot(ctx, root, func() error { return runner.Run(ctx, step.Command, emit) })
		} else {
			err = runner.Run(ctx, step.Command, emit)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", step.Name, err)
		}
	}
	if o.PublicKey != "" {
		if err = installKey(root, o.Username, o.PublicKey); err != nil {
			return err
		}
	}
	if o.ClearKeys {
		keys, _ := filepath.Glob(filepath.Join(root, "etc/ssh/ssh_host_*"))
		for _, key := range keys {
			if err = os.Remove(key); err != nil {
				return err
			}
		}
	}
	if o.ClearMachineID {
		if err = os.WriteFile(filepath.Join(root, "etc/machine-id"), nil, 0644); err != nil {
			return err
		}
		dbus := filepath.Join(root, "var/lib/dbus/machine-id")
		_ = os.Remove(dbus)
		if err = os.Symlink("/etc/machine-id", dbus); err != nil {
			return err
		}
	}
	if o.FirstBoot {
		return installFirstBoot(root, o.Hostname)
	}
	return nil
}
func withChroot(ctx context.Context, root string, run func() error) error {
	qemu, err := os.ReadFile("/usr/bin/qemu-aarch64-static")
	if err != nil {
		return err
	}
	dest := filepath.Join(root, "usr/bin/qemu-aarch64-static")
	if err = os.WriteFile(dest, qemu, 0755); err != nil {
		return err
	}
	defer os.Remove(dest)
	resolv := filepath.Join(root, "etc/resolv.conf")
	original, readErr := os.ReadFile(resolv)
	host, hostErr := os.ReadFile("/etc/resolv.conf")
	if hostErr == nil {
		_ = os.Remove(resolv)
		_ = os.WriteFile(resolv, host, 0644)
	}
	defer func() {
		if hostErr == nil {
			_ = os.Remove(resolv)
			if readErr == nil {
				_ = os.WriteFile(resolv, original, 0644)
			}
		}
	}()
	mounted := []string{}
	defer func() {
		for i := len(mounted) - 1; i >= 0; i-- {
			_ = runner.Run(context.Background(), runner.Command{Args: []string{"umount", mounted[i]}}, func(string) {})
		}
	}()
	for _, path := range []string{"proc", "sys", "dev", "dev/pts", "dev/shm"} {
		mountpoint := filepath.Join(root, path)
		if err = os.MkdirAll(mountpoint, 0755); err != nil {
			return err
		}
		args := []string{"mount", "--bind", "/" + path, mountpoint}
		if path == "proc" || path == "sys" {
			args = []string{"mount", "-t", path, path, mountpoint}
		}
		if err = runner.Run(ctx, runner.Command{Args: args}, func(string) {}); err != nil {
			return err
		}
		mounted = append(mounted, mountpoint)
	}
	return run()
}
func installKey(root, user, path string) error {
	key, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines, err := os.ReadFile(filepath.Join(root, "etc/passwd"))
	if err != nil {
		return err
	}
	var uid, gid int
	found := false
	for _, line := range strings.Split(string(lines), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) > 3 && fields[0] == user {
			_, err = fmt.Sscanf(fields[2]+" "+fields[3], "%d %d", &uid, &gid)
			found = err == nil
			break
		}
	}
	if !found {
		return fmt.Errorf("user not found in rootfs")
	}
	folder := filepath.Join(root, "home", user, ".ssh")
	if err = os.MkdirAll(folder, 0700); err != nil {
		return err
	}
	dest := filepath.Join(folder, "authorized_keys")
	old, _ := os.ReadFile(dest)
	if !strings.Contains(string(old), strings.TrimSpace(string(key))) {
		file, e := os.OpenFile(dest, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if e != nil {
			return e
		}
		_, err = io.WriteString(file, strings.TrimSpace(string(key))+"\n")
		_ = file.Close()
		if err != nil {
			return err
		}
	}
	_ = os.Chown(folder, uid, gid)
	return os.Chown(dest, uid, gid)
}

const firstBoot = `#!/bin/sh
set -eu
ssh-keygen -A
serial=$(tr -d '\000' </sys/firmware/devicetree/base/serial-number 2>/dev/null || true)
[ -n "$serial" ] || serial=$(cat /sys/class/net/eth0/address 2>/dev/null || true)
[ -n "$serial" ] || serial=$(od -An -N4 -tx1 /dev/urandom | tr -d ' ')
suffix=$(printf '%s' "$serial" | tr -cd 'a-zA-Z0-9' | tail -c 6 | tr '[:upper:]' '[:lower:]')
host=PREFIX-$suffix
hostnamectl set-hostname "$host"
sed -i '/^127\.0\.1\.1[[:space:]]/d' /etc/hosts
printf '127.0.1.1\t%s\n' "$host" >> /etc/hosts
mkdir -p /var/lib
touch /var/lib/jetson-firstboot.done
`

func installFirstBoot(root, prefix string) error {
	script := filepath.Join(root, "usr/local/sbin/jetson-firstboot.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(script, []byte(strings.Replace(firstBoot, "PREFIX", prefix, 1)), 0755); err != nil {
		return err
	}
	unit := `[Unit]
Description=Jetson first boot identity
DefaultDependencies=no
After=local-fs.target
Before=sockets.target ssh.socket ssh.service network-pre.target
ConditionPathExists=!/var/lib/jetson-firstboot.done
[Service]
Type=oneshot
ExecStart=/usr/local/sbin/jetson-firstboot.sh
[Install]
WantedBy=multi-user.target
`
	target := filepath.Join(root, "etc/systemd/system/jetson-firstboot.service")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(target, []byte(unit), 0644); err != nil {
		return err
	}
	link := filepath.Join(root, "etc/systemd/system/multi-user.target.wants/jetson-firstboot.service")
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		return err
	}
	_ = os.Remove(link)
	return os.Symlink("../jetson-firstboot.service", link)
}
