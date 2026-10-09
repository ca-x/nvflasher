# nvflasher

nvflasher v0.1.0 is a MyGo desktop app for preparing and flashing NVIDIA Jetson devices, with Orin Nano Super defaults. It wraps NVIDIA's official Linux for Tegra scripts; it does **not** implement a flashing protocol.

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

Provisioning creates a default user, optionally applies an overlay, installs APT packages inside chroot, and sends a non-interactive shell script to `chroot rootfs /bin/bash -s` via stdin. Script templates can be stored under the user config directory. Review scripts before execution. SSH forced Recovery requires a bootable Jetson, working SSH, and a previously trusted SSH host key in `~/.ssh/known_hosts`. HTTP(S) downloads accept direct URLs and optional HTTP, HTTPS, or SOCKS5 proxies; NVIDIA login is not supported.

Settings and non-secret flash/provisioning form values are saved in `os.UserConfigDir()/nvflasher/config.json`. Passwords and chroot script contents are not saved as last-used values (save an intentional script template instead). Proxy credentials are stored only when you explicitly select **Remember proxy credentials**; this may expose them to anyone who can access your account. Logs are redacted for common password arguments but avoid printing secrets in your custom scripts. Downloaded `.part` files are removed on cancel or error.

**Disclaimer:** NVIDIA BSP downloads and usage are subject to NVIDIA's EULA. Flashing can permanently destroy target data. Make backups and verify the target device, board configuration, and filesystem layout. Use at your own risk. Windows and macOS do not support flashing in this release.

License: MIT.
