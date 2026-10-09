# nvflasher — 项目规格书（SPEC）

> 本文件是 nvflasher 初版的实现规格。实现时的最高优先级参考。
> 语言约定：代码、注释、commit、README 主体用英文；本 SPEC 用中文（含英文术语）。

## 1. 项目定位

**nvflasher**：NVIDIA Jetson（重点 Orin Nano Super）刷机与预配置的桌面工具。

- 技术栈：MyGo（github.com/egoist/mygo，web 前端模板）+ Go 1.27+ + TypeScript/Vite 前端。
- 目的：把 NVIDIA 官方 `Linux_for_Tegra` 命令行流程（下载解压 → apply_binaries → 预置 rootfs → Recovery 刷机 → massflash 批量）做成一键式 GUI，减少手工操作与参数踩坑。
- 开源：MIT License，托管在 github.com/ca-x/nvflasher。
- 版本从 v0.1.0 开始。

## 2. 产品原则（必须遵守）

1. **诚实的能力分级**：不支持的平台上明确禁用相关功能并说明原因，绝不假装支持、绝不静默失败。功能可用性由运行时探测决定。
2. **不重复造轮子**：核心刷机逻辑包装 NVIDIA 官方工具链（调用 `l4t_initrd_flash.sh` 等），不重新实现刷机协议、不做 USB 底层 hack。
3. **不硬编码凭据与路径**：所有路径、密码从配置/表单来；密码不得写入日志与配置明文（配置里不建议存密码，SSH 密码仅存于本次会话内存）。
4. **命令构建与执行分离**：所有命令行由纯函数 builder 构造（可单测），执行统一走 runner（流式输出 + 可取消）。

## 3. 平台能力矩阵

| 功能 | Linux x86_64 | Windows | macOS |
| --- | --- | --- | --- |
| 环境检测（Linux_for_Tegra） | ✅ | ❌（禁用+说明） | ❌（禁用+说明） |
| Recovery USB 设备检测 | ✅（/sys） | ❌（禁用+说明） | ❌（禁用+说明） |
| 单台刷机 | ✅ | ❌ | ❌ |
| massflash 批量 | ✅ | ❌ | ❌ |
| rootfs 预置（用户/软件包） | ✅ | ❌ | ❌ |
| SSH 触发 Recovery（reboot forced-recovery） | ✅ | ✅ | ✅ |
| 镜像下载（含代理支持） | ✅ | ✅ | ✅ |

- 非 Linux 平台禁用项在 UI 上以分组灰显 + 提示条呈现：`Flashing requires a Linux x86_64 host (Linux_for_Tegra, USB, root). Not supported on this platform.`
- 判定用运行时探测（GOOS + 依赖存在性），不用编译期裁剪，同一代码库全平台可编译。
- 实现语言/注释里不要出现"TODO: support Windows flashing"这类暗示。

## 4. 功能需求

### F1 环境检测（Environment）

- 用户在设置里选择 `Linux_for_Tegra` 目录（文件选择对话框），路径持久化到配置文件。
- 检测项（每项给 ok/warn/error + 修复提示）：
  - 目录结构：`tools/kernel_flash/l4t_initrd_flash.sh`、`rootfs/` 存在
  - `apply_binaries` 已执行：`rootfs/etc/nv_tegra_release` 存在
  - BSP 版本：解析 `rootfs/etc/nv_tegra_release`（形如 `# R36 (release), REVISION: 4.3, ...`）
  - 板型列表：`ls Linux_for_Tegra/*.conf`（排除 common 文件），提取可用于 CLI 的 board 名（如 `jetson-orin-nano-devkit-super`）
  - 主机依赖（仅 Linux）：`qemu-aarch64-static`（存在且 `/proc/sys/fs/binfmt_misc` 有 aarch64 注册）、`sshpass`、`xmllint`、`dtc`、`nfs-server` 可用、`ssh-keygen`；缺失项给出发行版提示（Arch: `pacman -S ...`；Debian: `apt install ...`）
  - root 权限：`os.Geteuid()==0`；非 root 时顶部 banner 提示（说明 sudo/pkexec 启动；提供“以 root 重启”按钮：Linux 上 exec `pkexec <self>`，pkexec 不存在则提示手动）

### F2 设备检测（Devices）

- 直接读 `/sys/bus/usb/devices/*/idVendor` + `idProduct`（不依赖 lsusb 二进制）。
- 识别：
  - `0955:7X23`（X 任意 hex）→ Recovery 模式设备（T23x / Orin）显示为可刷机
  - `0955:7035` → initrd 模式（刷机进行中）
- 轮询（默认 2s），UI 实时显示设备列表与数量；支持多台（massflash 计数）。
- 非 Linux：面板显示"仅 Linux 支持"。

### F3 单台刷机（Flash）

- 表单字段（有合理默认值）：
  - board：下拉（来自 F1 conf 列表；默认 `jetson-orin-nano-devkit-super`；允许手动输入）
  - 内置"板型预设"：常见组合一键填充（如 "Orin Nano Super + NVMe" → board=jetson-orin-nano-devkit-super、external-device=nvme0n1p1、nvme xml、qspi xml；参考 balena jetson-flash 的设备字典做法）
  - external-device：默认 `nvme0n1p1`
  - nvme 配置：默认 `tools/kernel_flash/flash_l4t_t234_nvme.xml`
  - qspi 配置（-p 参数）：默认 `-c bootloader/generic/cfg/flash_t234_qspi.xml`
  - 开关：`--erase-all`（默认关，UI 警告数据会被清空）、`--showlogs`（默认开）
  - rootdev 固定 `internal`（说明文案：官方示例用法；NVMe 场景 internal/external 等价）
- 执行的命令形态（以 Linux_for_Tegra 为 cwd）：
  ```
  ./tools/kernel_flash/l4t_initrd_flash.sh --external-device <dev> \
    -c <nvme-xml> -p "-c <qspi-xml>" [--erase-all] [--showlogs] --network usb0 \
    <board> internal
  ```
- 前置检查：root 权限 + 至少一个 Recovery 设备（无设备时禁用按钮）。
- 流式日志到 UI；可取消（kill 进程组）；结束后给出成功/失败提示。

### F4 massflash 批量

- **生成 mfi 包**：
  - 表单同 F3 + 并发数 N（默认 5，说明：N 是并发上限，实际设备可少于 N）
  - 命令：同 F3 参数 + `--no-flash --massflash N`（不接设备也可以生成；说明离线模式可能需要 BOARDID 等环境变量）
  - 完成后显示产物 `mfi_<board>.tar.gz` 路径
- **批量刷写**：
  - 选择已解压的 mfi 目录，或选择 mfi 压缩包（工具自动解压到工作目录）
  - 命令（在 mfi 目录内）：`./tools/kernel_flash/l4t_initrd_flash.sh --flash-only --network usb0 --massflash N [--showlogs]`
  - 显示当前 Recovery 设备数量；提示所有设备需同硬件版本
- 两个流程都流式日志 + 可取消。

### F5 rootfs 预置（Provision）

- 前置：F1 环境 ok（已 apply_binaries）。
- 表单：
  - 用户名（默认 `dev`）、密码（默认随机生成 12 位，可显示/复制）、主机名占位（默认 `jetson`）
  - autologin 开关（`-a`）
  - SSH 公钥文件（文件选择器，可选）
  - 安装软件包（文本框，空格/换行分隔，可选）→ chroot apt
  - **chroot 定制脚本**（多行编辑器）：在 chroot 内以 root 执行的自定义 shell（非交互）；支持保存/加载模板（见下）
  - Overlay 目录（目录选择器，可选）
  - 清理项开关：删除 SSH 主机密钥（默认开）、清空 machine-id（默认开）
  - 安装 first-boot 服务（默认开）：首次开机生成主机密钥 + 按序列号设唯一主机名
- 执行步骤（顺序执行，任一步失败则停止；每步有独立日志）：
  1. 创建默认用户：`./tools/l4t_create_default_user.sh -u <user> -p <pass> -n <host> --accept-license [-a]`（**必须带 --accept-license**，否则 EULA 交互卡死；注意 `-a` 是 autologin）
  2. Overlay 拷贝：`cp -a <overlay>/. rootfs/`
  3. chroot 阶段（qemu + mount proc/sys/dev/dev-pts + /dev/shm；`trap` 确保卸载；备份并恢复 resolv.conf；结束时清理 qemu 二进制）：
     - 3a. 安装软件包：`apt-get update && apt-get install -y --no-install-recommends <PACKAGES>`（仅当 PACKAGES 非空）
     - 3b. chroot 定制脚本：把脚本经 stdin 传给 `chroot rootfs /bin/bash -s` 执行（不落盘）；注入 `DEBIAN_FRONTEND=noninteractive`；脚本必须非交互（UI 注明）
  4. SSH 公钥：写入 `rootfs/home/<user>/.ssh/authorized_keys`，用 `rootfs/etc/passwd` 里数字 uid/gid 设属主（幂等：先 grep 去重）
  5. 清理身份：删 `rootfs/etc/ssh/ssh_host_*`；`truncate -s 0 rootfs/etc/machine-id`；dbus machine-id 软链到 /etc/machine-id
  6. first-boot 服务：写 `rootfs/usr/local/sbin/jetson-firstboot.sh` + `rootfs/etc/systemd/system/jetson-firstboot.service` + `multi-user.target.wants` 软链；服务单元要求：
     - `DefaultDependencies=no`；`After=local-fs.target`；`Before=sockets.target ssh.socket ssh.service network-pre.target`
     - `ConditionPathExists=!/var/lib/jetson-firstboot.done`
     - 脚本：`ssh-keygen -A`；主机名 = `<prefix>-<serial 后 6 位>`（序列号读 `/sys/firmware/devicetree/base/serial-number`，失败退 eth0 MAC，再失败随机数）；同时更新 /etc/hosts 的 127.0.1.1 行；结束时 touch flag
- **chroot 模板**：
  - 用途：把常用 chroot 定制保存为可复用模板（装包组合、时区/locale、开发环境、预置自己的服务等）。
  - 内置只读模板（示例）：`base-packages`（curl/htop 等常用工具）、`timezone-locale`（Asia/Shanghai + en_US.UTF-8 示例）、`dev-tools`（git/build-essential/python3 等）。
  - 用户模板：保存为 `~/.config/nvflasher/templates/*.sh`（默认目录，界面可另行选择任意路径）；提供「加载 / 另存为 / 重命名 / 删除」；内容可直接编辑。
  - 执行前展示脚本全文供确认；执行输出并入日志面板。
  - 限制（UI 文案）：脚本在非交互环境执行，不要调用需要 TTY 的程序。
- 密码安全：不写入日志（命令回显中脱敏为 `***`）。

### F6 触发 Recovery（代码方式设置刷机模式）

- 目的：设备系统正常运行时，免手动跳线/按钮，通过 SSH 让设备重启进入 Recovery。
- 实现：Go SSH 客户端（golang.org/x/crypto/ssh，纯 Go 跨平台），执行 `sudo reboot forced-recovery`。
- 表单：地址（IP/主机名）、端口（默认 22）、用户名、SSH 密码、sudo 密码（默认同 SSH 密码，可分开填）。
- 流程：测试连接 → 执行命令 → UI 引导"等待设备以 Recovery 模式（0955:7523）出现"，与 F2 联动高亮。
- sudo 处理：优先 `sudo -n`（免密）测试；否则 `echo '<pass>' | sudo -S <cmd>`（密码经 stdin，不落入远端 shell history/进程参数）。
- 前提文案（UI 显著位置）：仅在设备系统可启动且 SSH 可达时有效；无系统/SSH 不可达的设备需要手动跳线（FC REC+GND）或按钮（RECOVERY+RESET）。
- 该功能所有平台可用（不依赖 Linux_for_Tegra）。

### F7 镜像下载（Download）

- 目的：在 GUI 内下载 Jetson Linux BSP / Sample Root Filesystem 等大镜像文件，**支持通过代理下载**（面向需要代理访问 NVIDIA 下载源的网络环境）。
- 表单：
  - URL（可粘贴任意直链；内置 Jetson Linux 归档页 / JetPack 下载页入口链接与常用 BSP 直链参考列表（来自官方归档，可能要求登录或失效，仅作参考），并说明：部分官方链接要求 NVIDIA 账号登录，nvflasher 不做登录，请提供可访问的直链）
  - 保存目录（默认 `~/Downloads` 或上次使用目录）
  - 文件名（默认从 URL 推断，可改）
  - 可选：SHA256 校验值（提供则下载后校验）
- 代理：
  - 默认读取 Settings 中的代理配置；面板内可临时覆盖
  - 支持 `http://host:port`、`https://host:port`、`socks5://host:port`，以及带认证形式 `http://user:pass@host:port`
  - "测试代理"按钮：对目标地址发 HEAD 请求验证连通性
- 下载行为：流式进度（百分比/字节/速度/ETA）通过 mygo Channel 推送到 UI；可取消（清理 `.part` 临时文件）；服务器支持 Range 时尽量断点续传。
- 完成后：显示文件路径与大小；提供"在文件管理器中显示"（跨平台 opener：xdg-open / open / explorer）。
- 平台：全平台可用（不依赖 Linux_for_Tegra）。

### F8 设置（Settings）

- 持久化（JSON，`os.UserConfigDir()/nvflasher/config.json`）：L4T 路径、工作目录（下载 / mfi 产物 / 日志的存放位置）、上次使用的表单值、日志行数上限、下载目录等。
- 网络代理配置：`proxy.enabled`、`proxy.url`（支持 http/https/socks5）；代理认证信息默认不落盘（提供"记住代理凭据"复选框，默认不勾；勾选才写入配置，README 提示风险）。
- 不存其他任何密码（SSH/sudo 密码仅存于会话内存）。

### F9 日志与运行器（贯穿）

- runner：`exec.CommandContext` + `Setpgid`；stdout/stderr 按行合并流式推送（mygo Channel）；支持取消（给进程组发 SIGTERM，超时后 SIGKILL）。
- 日志缓冲：内存保留最近 N 行（默认 5000）；提供"保存日志到文件"。
- UI：日志面板支持自动滚动、暂停滚动、复制。

## 5. 技术架构

```
main.go                      — mygo App 初始化、窗口、Bind services、App.WhenReady
internal/
  capability/                — 平台与依赖能力探测（GOOS 探测 + 文件/命令存在性）
  config/                    — 配置读写（JSON、UserConfigDir）
  tegra/                     — L4T 环境解析：目录校验、版本解析、board conf 列表、命令 builder（纯函数）
  runner/                    — 子进程执行、流式输出、取消、脱敏
  provision/                 — F5 各步骤编排（生成 shell 片段 + 调 runner；步骤可测）
  flash/                     — F3 命令构建与执行编排
  massflash/                 — F4 生成/刷写编排
  usbwatch/                  — /sys/bus/usb 轮询解析（解析函数纯化、可测，注入 sysfs 根路径）
  sshx/                      — F6 SSH 执行（接口化，便于测试）
frontend (src/)              — TS + Vite；用 mygo 生成的 typed client；channel 接收日志
```

- mygo 绑定服务建议（命名固定，前端由此生成 client）：
  - `Environment`（detect, getConfig, saveConfig）
  - `Devices`（startWatch/stopWatch，事件 devices:changed）
  - `Flash`（start（channel 输出日志）、cancel）
  - `Massflash`（generate、flash、cancel）
  - `Provision`（run（channel 输出）、cancel、generatePassword；chroot 脚本执行 + 模板管理：listTemplates / saveTemplate / loadTemplate）
  - `Recovery`（trigger（SSH））
  - `Download`（start（channel 输出进度）、cancel、testProxy）
- 结构体字段与 JSON tag 明确，用 mygo 支持的返回类型（error 单独返回）。
- 前端 UI 结构：左侧导航（Environment / Devices / Flash / Massflash / Provision / Recovery / Download / Settings），主区域 + 底部日志抽屉。样式简洁、深色友好（不追求华丽）。所有危险操作（--erase-all、刷机启动、chroot 脚本执行）需二次确认对话框。

## 6. 质量与交付

- `gofmt`、`go vet` 干净；`go test ./...` 通过（tegra builder、usbwatch 解析、config、provision 步骤生成 有单测）。
- 前端 `bun run build` 成功；TS 无类型错误。
- `mygo build` 在本机产出 Linux 包；`mygo build -platform darwin/universal,windows/amd64,linux/amd64` 交叉编译通过（至少保证编译）。
- `README.md`（英文为主）+ `README.zh-CN.md`（中文）：项目简介、功能矩阵（同第 3 节表格）、截图占位、安装/构建、开发（mygo dev）、使用前提、安全与免责声明（刷机风险自负；需要 Linux 主机）。
- `LICENSE`：MIT（作者写 "ca-x" 组织或按 generate 默认）。
- `.github/workflows/ci.yml`：ubuntu-latest；setup-go、setup-bun、`go test ./...`、`bun install && bun run build`（或等价的 mygo build 验证）。
- git：`git init`，首个 commit 用 conventional commit（`feat: initial nvflasher release`）。**不要 push 到任何 remote**（由外部完成）。

## 7. 明确不做（Non-goals / v1 排除）

- Windows/macOS 上执行刷机（不硬做；只做禁用+说明）
- 远程后端模式（GUI 连远程 Linux 刷机机）：不在 v1
- 制作 JetPack ISO 启动盘 / 写 SD 卡镜像
- NVIDIA 账号登录、抓取下载页、解析官方下载接口：下载功能只支持用户可访问的直链（可走代理），不做登录流程
- JetPack 5.x 及更早版本的兼容承诺
- 自定义载板/安全启动/加密刷机的高级流程

## 8. 已知技术细节（避免踩坑）

- mygo：`mygo dev` 开发；`mygo generate` 生成 src/mygo.ts；channel 用于流式输出（见 mygo 文档 "Calling Go from the frontend"）。
- 项目已由 `mygo init`（v0.3.5 web 模板）生成于当前目录；开始前先读 `.agents/skills/mygo-maintenance/SKILL.md`（mygo 官方给的维护指引）。
- 本机环境：Go 1.27.1（mise）、bun 1.4.2（/home/czyt/.local/bin/bun）、mygo CLI 在 PATH；Arch Linux；GTK3 + WebKitGTK 4.1 已装；npm 直连可用。
- 常用命令：`bun run typecheck`、`bun run build`（= mygo build，产出 build/ 下的包）、`go test ./...`。
- mygo.config.ts：identifier 改为 `tech.czyt.nvflasher`，version 保持 0.1.0；窗口标题 `nvflasher`。
- 不要尝试在有显示依赖的情况下跑 GUI 自动化测试；构建与 `go test` 即可。
- `l4t_create_default_user.sh` 位置：`<L4T>/tools/`；`apply_binaries.sh` 在 `<L4T>/` 根。

## 9. 参考项目与借鉴（balena-os/jetson-flash）

参考 balena 的 jetson-flash（Node.js CLI + Docker，Apache-2.0）的以下设计：

1. **设备字典驱动**：每个设备一个配置（BSP 直链、分区映射、刷写介质）→ nvflasher 采纳为 F3 的"板型预设"（内置几条常见组合的默认参数）。
2. **下载 + 缓存 BSP**：设备字典带 BSP URL、支持 `--cache` → nvflasher 的 F7 内置常用直链参考列表；工作目录用于复用下载产物。
3. **工作目录管理**：`-o/--persistent` 控制产物位置 → nvflasher 的 workspace 配置（F8）。
4. **许可确认流程**：刷机前要求接受 NVIDIA EULA → nvflasher 以 README 免责声明 + 首次运行提示体现（不做阻塞式弹窗）。
5. **Docker 隔离宿主依赖**（Dockerfile 打包工具链，宿主只需 Docker + USB）→ 列为未来方向，不在 v1 范围（Linux 宿主本地运行已足够）。

注意：jetson-flash 面向 balenaOS 镜像刷写，nvflasher 面向完整 L4T BSP 流程；借鉴的是流程与配置组织方式，代码不通用。
- mfi 产物：`mfi_<board>.tar.gz`（不是 .tbz2）。
- `reboot forced-recovery` 需要设备侧 sudo；若无免密则用 `sudo -S` 从 stdin 喂密码。
- 日志脱敏：任何命令回显把 `-p <password>`/`sudo -S` 相关内容遮蔽。
