# nvflasher

<p align="center"><img src="assets/logo.png" alt="nvflasher" width="400"></p>

nvflasher v0.1.1 is a MyGo desktop app for preparing and flashing NVIDIA Jetson devices, with Orin Nano Super defaults. It wraps NVIDIA's official Linux for Tegra scripts; it does **not** implement a flashing protocol.

| Feature | Linux x86_64 | Windows / macOS |
| --- | --- | --- |
| BSP diagnostics, Recovery USB discovery | Yes | Disabled |
| Single flash, MFI generation/flash | Yes, root required | Disabled |
| Rootfs provisioning, packages, chroot scripts | Yes, root required | Disabled |
| SSH forced recovery, direct image download | Yes | Yes |

Screenshot: _to be added after first packaged release_.

## Build

Requires Go 1.27.2+, Bun, MyGo CLI 0.3.5, and the MyGo GTK3/WebKitGTK 4.1 build prerequisites on Linux.

```sh
bun install
go test ./...
bun run typecheck
bun run build
```

The Linux bundle is placed in `build/`. To work on the UI, run `mygo dev` in a graphical session. Cross-platform builds depend on the corresponding MyGo packaging toolchains.

## Build and release

On Linux, build both installation packages with:

```sh
bun run build -- -platform linux/amd64
bun run build -- -platform linux/arm64
```

Each `build/linux-<arch>/` directory contains a `.deb` and `.tar.gz` package. GitHub Actions uploads them as `nvflasher-linux-amd64` and `nvflasher-linux-arm64` artifacts after every CI build (including manual `workflow_dispatch` runs). Pushing a `v*` tag also publishes both architectures' packages to a GitHub Release after tests pass.

## Usage and safety

On a Linux x86_64 host, extract the NVIDIA BSP and sample rootfs into `Linux_for_Tegra`, run `apply_binaries.sh`, and provide its directory under Settings. Install host dependencies as reported by Environment. Start nvflasher with root permissions (`pkexec` or `sudo`) when flashing or provisioning; connect the Jetson via USB in forced Recovery mode. The app only executes NVIDIA's official tools. For MFI flash, supply an extracted MFI directory or a `.tar.gz` archive (safely extracted into the workspace); MFI generation produces `mfi_<board>.tar.gz` in the BSP directory. Only identical hardware should be flashed together.

Use the language menu to switch between English and Simplified Chinese; the selection is stored locally. On Linux, the Environment page can accept a sudo password to restart with root permissions. The password is used once for authentication, never written to configuration or logs. The elevated app may use separate root-user settings; verify the BSP path and flash options again after restarting.

After an interrupted BSP preparation, retry “Extract and prepare BSP” in the Download step. The app reuses verified archives and extracted files, removing only leftover rootfs character-device nodes that match expected host device numbers before retrying `apply_binaries.sh`; it never deletes regular files or symlinks there. UI sudo restart requires a working systemd user session; otherwise use `pkexec`.

Environment can list missing packages on Debian, Ubuntu, and Arch derivatives (including Omarchy and EndeavourOS), copy the installation command, or install them after confirmation in a root session. On Arch, BSP preparation needs the host `dpkg` package, now included in missing-dependency checks. Arch installation runs `pacman -Syu`, which upgrades the whole system. NVIDIA does not certify Arch as a Jetson Linux flashing host; installing these packages does not guarantee flashing works on Arch.

Flashing also requires the `abootimg` executable on the host. Debian/Ubuntu can install it through the Environment page. On Arch the `abootimg` package is in the AUR rather than the official pacman repositories; the UI offers a copyable `paru -S abootimg` command to run manually as a regular user, never as root. Rerun diagnostics after installing it.

On Arch, flashing maps NVIDIA's `service nfs-kernel-server` calls to systemd's `nfs-server.service` only inside the flash subprocess; it does not change the global host commands. Install `nfs-utils` first. Destructive operations and package installation use native MyGo confirmation dialogs; template names use an inline field instead of browser prompts.

The UI groups tasks into Flash setup and Rootfs setup wizards. Flash setup walks through environment checks, model selection, official downloads and BSP preparation, and flashing. The automatic catalog covers Jetson Orin Nano 8GB and Super Developer Kits with Jetson Linux 36.5.0: they share downloads but use different flash board configurations. The BSP contains NVIDIA's flashing tools; the app verifies both archives against NVIDIA SHA1 checksums before extraction and runs `apply_binaries.sh` with root privileges. A USB ID alone does not prove the board model; verify the device label. The flash step offers a prebuilt MFI directory or `.tar.gz` archive as a separate mode. An arbitrary ISO or `.img` file is not an MFI package.

Provisioning shows the selected `Linux_for_Tegra/rootfs` directory and allows browsing to another BSP. It can create a user, apply an overlay, install APT packages inside chroot, and send a non-interactive script to `chroot rootfs /bin/bash -s`. Select multiple templates and add their scripts to the visible editor before running; review scripts before execution. SSH forced Recovery requires a bootable Jetson, working SSH, and a previously trusted SSH host key in `~/.ssh/known_hosts`. Official package downloads accept optional HTTP, HTTPS, or SOCKS5 proxies; NVIDIA login is not supported.

Settings and non-secret flash/provisioning form values are saved in `os.UserConfigDir()/nvflasher/config.json`. Passwords and chroot script contents are not saved as last-used values (save an intentional script template instead). Proxy credentials are stored only when you explicitly select **Remember proxy credentials**; this may expose them to anyone who can access your account. Logs are redacted for common password arguments but avoid printing secrets in your custom scripts. Downloaded `.part` files are removed on cancel or error.

**Disclaimer:** NVIDIA BSP downloads and usage are subject to NVIDIA's EULA. Flashing can permanently destroy target data. Make backups and verify the target device, board configuration, and filesystem layout. Use at your own risk. Windows and macOS do not support flashing in this release.

License: MIT.
