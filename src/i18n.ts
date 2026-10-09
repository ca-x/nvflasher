export type Language = "en" | "zh-CN";

export const messages: Record<string, string> = {
  "Language": "语言", "JETSON WORKSTATION · v0.1.0": "JETSON 工作站 · v0.1.0",
  "Official NVIDIA toolchain wrapper": "NVIDIA 官方工具链界面",
  "Environment": "环境检测", "Devices": "设备", "Flash": "刷写", "Massflash": "批量刷写",
  "Flash setup": "刷机向导", "Rootfs setup": "Rootfs 配置向导", "Step": "步骤", "of": "/",
  "Back": "上一步", "Next step": "下一步", "Back to wizard": "返回向导", "Global settings": "全局设置",
  "Batch flashing": "批量刷写", "Single-device flash": "单机刷写", "Back to devices": "返回设备页",
  "Use a prebuilt MFI package": "使用预生成 MFI 包", "Flash MFI": "刷写 MFI",
  "Flash a prebuilt MFI package or generate a new one from the BSP.": "刷写预生成的 MFI 包，或使用 BSP 生成新包。",
  "Prebuilt MFI package": "已有 MFI 包", "MFI directory or .tar.gz archive": "MFI 目录或 .tar.gz 压缩包",
  "Maximum devices": "设备数量上限", "Only flash devices matching the MFI hardware. Recovery USB is required.": "仅可刷写与 MFI 硬件型号匹配的设备，且需要恢复模式 USB。",
  "Generate new MFI package": "生成新的 MFI 包", "Generate a new MFI package?": "生成新的 MFI 包吗？",
  "Provision": "预置", "Recovery": "恢复模式", "Download": "下载", "Settings": "设置",
  "RUN LOG": "运行日志", "Ready": "就绪", "Running": "运行中", "Completed": "已完成", "Failed": "失败",
  "Auto-scroll": "自动滚动", "Copy": "复制", "Save": "保存", "Hide": "隐藏", "Show": "显示",
  "LINUX x86_64 · ROOT READY": "LINUX x86_64 · 已取得 ROOT 权限",
  "LINUX x86_64 · ROOT REQUIRED": "LINUX x86_64 · 需要 ROOT 权限",
  "NON-LINUX FLASHING DISABLED": "非 Linux 系统无法刷写",
  "recovery devices": "台恢复模式设备",
  "Inspect your Linux for Tegra BSP and host dependencies.": "检查 Linux for Tegra BSP 和主机依赖。",
  "Flashing requires a Linux x86_64 host (Linux_for_Tegra, USB, root). Not supported on this platform.": "刷写需要 Linux x86_64 主机（Linux_for_Tegra、USB 和 root）。当前平台不支持。",
  "Root required: restart with pkexec or sudo to enable flashing.": "刷写需要 root 权限。请使用系统授权或输入 sudo 密码重新启动。",
  "Linux_for_Tegra directory": "Linux_for_Tegra 目录", "Browse": "浏览", "Run diagnostics": "运行诊断",
  "Restart as root": "系统授权后重启", "Sudo password": "sudo 密码", "Restart with sudo": "输入密码并提权重启",
  "Password is used once for sudo authentication and is not saved.": "密码仅用于本次 sudo 验证，不会保存。",
  "Run diagnostics to inspect BSP.": "运行诊断以检查 BSP。",
  "Recovery USB devices refresh every two seconds on Linux.": "Linux 上的恢复模式 USB 设备每两秒刷新一次。",
  "Connected Jetson devices refresh every two seconds on Linux. Flashing requires Recovery mode.": "Linux 上的 Jetson 设备每两秒刷新一次；刷写需要进入恢复模式。",
  "No NVIDIA Jetson devices found.": "未找到 NVIDIA Jetson 设备。",
  "Connected NVIDIA USB devices refresh every two seconds on Linux. Flashing requires Recovery mode.": "Linux 上的 NVIDIA USB 设备每两秒刷新一次；刷写需要进入恢复模式。",
  "No NVIDIA USB devices found.": "未找到 NVIDIA USB 设备。",
  "L4T-README board IDs": "L4T-README 中的板卡 ID",
  "No Recovery device detected. Enter Recovery before flashing; a running device cannot be flashed.": "未检测到恢复模式设备。正常运行的设备不能刷写，请先进入 Recovery 模式。",
  "Checking BSP readiness…": "正在检查 BSP 准备状态…", "BSP preparation incomplete": "BSP 尚未准备完成",
  "Continue BSP preparation": "继续准备 BSP", "Preparation in progress; wait for completion": "BSP 准备进行中，请等待完成",
  "Enter Recovery via SSH": "通过 SSH 进入 Recovery",
  "Rootfs to configure": "将要配置的 rootfs", "No prepared rootfs selected": "尚未选择已准备的 rootfs",
  "Prepare rootfs in Flash wizard": "前往刷机向导准备 rootfs",
  "Script templates": "脚本模板", "Select multiple templates, then add their scripts above. Only the visible script runs when you provision.": "可选多个模板并添加到上方脚本。预置时只执行脚本框中可见的内容。",
  "Add selected scripts": "添加所选脚本", "Selected templates": "已选模板", "No templates selected": "尚未选择模板",
  "Scripts added to the editor": "脚本已加入编辑框", "Waiting to download or prepare": "等待下载或准备",
  "Extracting NVIDIA BSP (includes flash tools)": "正在解压 NVIDIA BSP（含刷机工具）", "Extracting sample rootfs": "正在解压示例 rootfs",
  "Reusing extracted NVIDIA BSP": "复用已解压的 NVIDIA BSP", "Reusing extracted sample rootfs": "复用已解压的示例 rootfs",
  "Applying NVIDIA binaries": "正在应用 NVIDIA 二进制组件",
  "Select device model": "选择设备型号", "Choose model…": "请选择型号…",
  "USB ID alone does not identify the board. Confirm the model from the device label before flashing.": "仅靠 USB ID 无法识别板卡，请核对设备标签后再刷机。",
  "Download the official BSP (including flash tools) and rootfs for the selected model.": "根据所选型号下载官方 BSP（含刷机工具）和 rootfs。",
  "Choose a model on the Devices step first": "请先在设备步骤选择型号",
  "Archives are verified with NVIDIA SHA1 hashes before extraction. Preparation requires root and can use substantial disk space.": "解压前会校验 NVIDIA SHA1；准备过程需要 root 且占用大量磁盘空间。",
  "Download BSP, flash tools and rootfs": "下载 BSP、刷机工具及 rootfs", "Extract and prepare BSP": "解压并准备 BSP",
  "BSP ready": "BSP 已准备就绪", "Download complete. Restart with sudo, then select Extract and prepare BSP.": "下载完成。请用 sudo 重新启动，再选择“解压并准备 BSP”。",
  "Checking package requirements…": "正在检查软件依赖…", "System dependencies": "系统依赖",
  "No missing packages detected.": "未检测到缺失的软件包。", "Missing packages": "缺失的软件包",
  "Arch installation updates the entire system. NVIDIA does not certify Arch as a flashing host.": "Arch 安装会升级整个系统。NVIDIA 不认证 Arch 为刷写主机。",
  "Package installation changes the host system.": "安装软件包会修改主机系统。",
  "Copy install command": "复制安装命令", "Install missing packages": "安装缺失的软件包",
  "Install these packages on the host?": "在主机上安装这些软件包吗？",
  "Unknown version": "未知版本", "boards": "个板型",
  "BSP not prepared — choose a model and download on the previous steps": "BSP 尚未准备：请选择型号并在下载页完成解压与准备",
  "flash tool": "刷写工具", "ok": "正常", "error": "错误", "warn": "警告",
  "Missing": "缺少", "extract BSP/rootfs and run apply_binaries.sh": "请解压 BSP/rootfs 并运行 apply_binaries.sh",
  "Install": "安装", "using your distribution package manager": "请使用发行版包管理器",
  "Register qemu-aarch64 with binfmt_misc": "请向 binfmt_misc 注册 qemu-aarch64",
  "Install nfs-utils (Arch) or nfs-kernel-server (Debian)": "请安装 nfs-utils（Arch）或 nfs-kernel-server（Debian）",
  "running": "正常运行", "recovery": "恢复模式", "initrd": "initrd 模式",
  "No Recovery or initrd devices found.": "未找到恢复模式或 initrd 设备。",
  "Flash device": "刷写设备", "Generate MFI offline or flash matching Recovery devices in parallel.": "离线生成 MFI，或并行刷写匹配的恢复模式设备。",
  "Run NVIDIA's official initrd flashing tool for one Recovery device.": "对一台恢复模式设备运行 NVIDIA 官方 initrd 刷写工具。",
  "Target configuration": "目标配置", "Board preset": "板型预设", "Custom": "自定义", "Board": "板型",
  "External device": "外部存储设备", "NVMe layout XML": "NVMe 分区 XML", "QSPI layout XML": "QSPI 分区 XML",
  "Erase all storage": "擦除全部存储", "Detailed logs": "详细日志",
  "rootdev: internal · network: usb0 · Erase all destroys target data.": "rootdev: internal · network: usb0 · 全盘擦除会删除目标数据。",
  "Concurrency limit": "并发上限", "Extracted MFI directory": "已解压的 MFI 目录",
  "Offline generation may require BOARDID. Output: mfi_<board>.tar.gz. All devices must be identical hardware.": "离线生成可能需要 BOARDID。产物：mfi_<board>.tar.gz。所有设备必须使用相同硬件。",
  "Generate MFI": "生成 MFI", "Start flash": "开始刷写", "Flash MFI devices": "刷写 MFI 设备", "Cancel": "取消",
  "Provision rootfs": "预置 rootfs", "Create a user, install packages and customize rootfs using a non-interactive chroot.": "通过非交互式 chroot 创建用户、安装软件包并自定义 rootfs。",
  "Scripts run as root with no TTY. Review the entire script before running it.": "脚本以 root 身份运行且没有 TTY。运行前请检查完整脚本。",
  "Username": "用户名", "Hostname prefix": "主机名前缀", "Password": "密码", "Generate": "生成",
  "SSH public key": "SSH 公钥", "Overlay directory": "Overlay 目录", "Autologin": "自动登录",
  "APT packages (space/newline separated)": "APT 软件包（空格或换行分隔）",
  "Chroot script (stdin, non-interactive)": "Chroot 脚本（标准输入，非交互式）",
  "Script template": "脚本模板", "Choose…": "请选择…", "Load": "加载", "Save as": "另存为",
  "Rename": "重命名", "Delete": "删除", "Clear SSH host keys": "清除 SSH 主机密钥",
  "Clear machine-id": "清除 machine-id", "First-boot identity service": "首次启动身份服务",
  "Run provisioning": "执行预置", "built-in": "内置",
  "SSH Recovery": "通过 SSH 进入恢复模式", "Reboot a running Jetson into forced recovery over SSH. Available on all platforms.": "通过 SSH 将运行中的 Jetson 重启到强制恢复模式。所有平台均可使用。",
  "Requires a bootable device, SSH access and a trusted host key in ~/.ssh/known_hosts. Otherwise use FC REC + GND or RECOVERY + RESET.": "需要设备可启动、SSH 可访问，且主机密钥已保存在 ~/.ssh/known_hosts。否则请使用 FC REC + GND 或 RECOVERY + RESET。",
  "Host/IP": "主机/IP", "Port": "端口", "SSH password": "SSH 密码",
  "Sudo password (optional)": "sudo 密码（可选）", "Request forced recovery": "请求强制恢复模式",
  "Download images": "下载镜像", "Use a direct HTTP(S) link. Some NVIDIA sources require login, which this app does not provide.": "请使用 HTTP(S) 直链。某些 NVIDIA 下载地址需要登录，本软件不提供登录功能。",
  "Direct URL": "直链地址", "Save directory": "保存目录", "Filename (optional)": "文件名（可选）",
  "Proxy override (HTTP/HTTPS/SOCKS5)": "单次代理（HTTP/HTTPS/SOCKS5）",
  "Expected SHA256 (optional)": "预期 SHA256（可选）",
  "Show in file manager": "在文件管理器中显示", "Choose archive": "选择压缩包",
  "Test proxy to URL": "通过代理测试此地址", "Target reachable": "目标地址可访问",
  "Saved locally. SSH, sudo and device passwords are never stored.": "设置保存在本机；SSH、sudo 和设备密码不会保存。",
  "Workspace": "工作目录", "Download directory": "下载目录", "Maximum log lines": "日志行数上限",
  "Enable proxy": "启用代理", "HTTP/HTTPS/SOCKS5 proxy URL": "HTTP/HTTPS/SOCKS5 代理地址",
  "Remember proxy credentials (stored on disk; security risk)": "记住代理凭据（写入磁盘，存在安全风险）",
  "Save settings": "保存设置", "Test proxy": "测试代理", "Settings saved": "设置已保存", "Proxy reachable": "代理可访问",
  "Operation completed": "操作完成", "This operation may change or erase device data. Continue?": "此操作可能修改或擦除设备数据。继续吗？",
  "ERASE ALL enabled. Wipe the connected Jetson?": "已启用全盘擦除。要清空连接的 Jetson 吗？",
  "Start flashing the connected Jetson?": "开始刷写连接的 Jetson 吗？",
  "Flash all connected matching devices?": "刷写所有已连接且型号匹配的设备吗？",
  "Run rootfs provisioning?": "执行 rootfs 预置吗？", "Full chroot script:": "完整 chroot 脚本：",
  "(none)": "（无）", "Reboot the device into Recovery mode?": "将设备重启到恢复模式吗？",
  "Enter Recovery mode and troubleshoot": "短接进入恢复模式与故障排查",
  "Recovery guide": "恢复模式指南",
  "These jumper steps apply to the official Jetson Orin Nano Developer Kit. Check your carrier board manual if its button header differs.": "以下短接步骤适用于官方 Jetson Orin Nano 开发套件；若载板排针不同，请先查阅该载板手册。",
  "Disconnect the power cable and connect the data USB cable to the host.": "拔掉设备电源，将可传输数据的 USB 线接到主机。",
  "On the 12-pin button header, short the pins labeled REC and GND. Do not guess pin positions.": "在 12 针按钮排针上短接标为 REC 和 GND 的针脚；不要猜测针脚位置。",
  "Reconnect power while REC and GND are shorted. After the host detects Recovery USB, remove the jumper.": "保持 REC 与 GND 短接，重新接通电源；主机检测到恢复模式 USB 后移除短接线。",
  "Troubleshooting": "故障排查",
  "No NVIDIA USB device: check power, the data-capable USB cable, and the host USB port; then repeat the power-off sequence.": "未发现 NVIDIA USB 设备：检查供电、数据 USB 线和主机 USB 接口，然后断电重试。",
  "A running-mode device cannot be flashed: enter Force Recovery and wait for the USB device list to refresh.": "设备处于正常运行模式时不能刷写：进入强制恢复模式并等待 USB 列表刷新。",
  "For the Orin Nano 8GB module, 0955:7523 (APX) indicates Force Recovery; a stopped fan alone does not identify the mode.": "Orin Nano 8GB 模组出现 0955:7523（APX）表示进入强制恢复模式；不能仅凭风扇不转判断状态。",
  "Recovery USB is present but flashing is unavailable: complete BSP preparation on the Download step and rerun Environment diagnostics.": "已发现恢复模式 USB 但无法刷写：在“下载”步骤完成 BSP 准备，并重新运行环境检测。",
  "USB ID alone does not verify the board model. Confirm the hardware label and selected model before flashing.": "USB ID 无法单独确认载板型号。刷写前请核对硬件标签和选定型号。",
  "NVIDIA recovery instructions": "NVIDIA 官方恢复模式说明",
  "Template name": "模板名称", "New name": "新名称",
  "NVIDIA BSP is subject to NVIDIA EULA. Flashing may erase data. Proceed at your own risk.": "NVIDIA BSP 受 NVIDIA EULA 约束。刷写可能清除数据，请自行承担风险。",
};

export function translate(text: string, language: Language): string {
  return language === "zh-CN" ? messages[text] ?? text : text;
}

export function translateCheck(status: string, hint: string, language: Language): string {
  if (language === "en") return `${status}: ${hint}`;
  if (hint.startsWith("Missing ") && hint.endsWith("; extract BSP/rootfs and run apply_binaries.sh")) {
    const path = hint.slice("Missing ".length, -"; extract BSP/rootfs and run apply_binaries.sh".length);
    return `${translate(status, language)}: ${translate("Missing", language)} ${path}；${translate("extract BSP/rootfs and run apply_binaries.sh", language)}`;
  }
  if (hint.startsWith("Missing ") && hint.endsWith("; run BSP preparation in the Download step")) {
    const path = hint.slice("Missing ".length, -"; run BSP preparation in the Download step".length);
    return `${translate(status, language)}: ${translate("Missing", language)} ${path}；请在“下载”步骤点击“解压并准备 BSP”重试`;
  }
  if (hint.startsWith("Install ") && hint.endsWith(" using your distribution package manager")) {
    const name = hint.slice("Install ".length, -" using your distribution package manager".length);
    return `${translate(status, language)}: ${translate("Install", language)} ${name}，${translate("using your distribution package manager", language)}`;
  }
  return `${translate(status, language)}: ${translate(hint, language)}`;
}

const sourceText = new WeakMap<Node, string>();

export function localize(root: Node, language: Language): void {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  while (walker.nextNode()) {
    const node = walker.currentNode;
    if (node.parentElement?.closest("#log, select#language")) continue;
    const text = node.textContent ?? "";
    if (!sourceText.has(node)) sourceText.set(node, text);
    const original = sourceText.get(node)!;
    const trimmed = original.trim();
    if (trimmed && messages[trimmed]) node.textContent = original.replace(trimmed, translate(trimmed, language));
  }
}
