# nvflasher

nvflasher v0.1.0 基于 MyGo 构建，重点支持 Jetson Orin Nano Super。它调用 NVIDIA 官方 Linux for Tegra 脚本，不自行实现刷机协议。

| 功能 | Linux x86_64 | Windows / macOS |
| --- | --- | --- |
| BSP 环境检测、Recovery USB 识别 | 支持 | 禁用 |
| 单机刷写、MFI 生成与批量刷写 | 需要 root | 禁用 |
| rootfs 用户预置、装包、chroot 自定义脚本 | 需要 root | 禁用 |
| SSH 触发 Recovery、直链下载 | 支持 | 支持 |

截图：_首次发布后补充_。

## 构建与开发

需要 Go 1.27.2+、Bun、MyGo CLI 0.3.5；Linux 构建另需 GTK3 / WebKitGTK 4.1 开发环境。

```sh
bun install
go test ./...
bun run typecheck
bun run build
```

生成的 Linux 程序包位于 `build/`。图形环境下可用 `mygo dev` 开发。跨平台构建需要目标平台对应的 MyGo 工具链。

## 使用与风险

在 Linux x86_64 主机上下载并解压 NVIDIA BSP、sample rootfs，执行 `apply_binaries.sh`，在设置中指定 `Linux_for_Tegra` 目录；按环境检测提示安装依赖。刷机或预置时通过 `pkexec` 或 `sudo` 以 root 权限启动；确保设备通过 USB 进入 Recovery 模式。批量刷机可选择已解压的 MFI 目录或 `.tar.gz` 压缩包（安全解压到工作目录），并确保设备硬件版本一致。

预置功能支持用户、overlay、chroot apt 装包、通过 stdin 执行 `chroot rootfs /bin/bash -s` 非交互脚本；支持模板保存和加载。执行前务必检查完整脚本。SSH Recovery 需要设备可开机、SSH 可用，且主机密钥已加入 `~/.ssh/known_hosts`。下载仅支持可访问的 HTTP(S) 直链与 HTTP/HTTPS/SOCKS5 代理，不支持 NVIDIA 帐号登录。

设置写入 `os.UserConfigDir()/nvflasher/config.json`。设备和 SSH 密码不持久化；只有勾选“记住代理凭据”才会保存代理账户信息（存在泄露风险）。自定义脚本应避免输出敏感数据。

**免责声明：**使用 NVIDIA BSP 须遵守 NVIDIA EULA。刷机可能永久清除设备数据；请备份并核对目标设备及板型。风险由使用者承担。Windows/macOS 不支持刷机。

许可证：MIT。
