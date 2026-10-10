import { Channel } from "mygo-runtime";
import { localize, translate, translateCheck, type Language } from "./i18n";
import { activeStep, steps, type Page, type Workflow } from "./navigation";
import { appendTemplateScripts } from "./provision-templates";
import { matchingPresetName } from "./presets";
import { Devices, Download, Environment, Flash, Massflash, Provision, Recovery, Setup, events, type Config, type Device, type Model, type Options, type ProvisionOptions, type Status, type Template } from "./mygo";
const $ = <T extends HTMLElement>(selector: string) => document.querySelector<T>(selector)!;
let workflow: Workflow = "Flash";
let page: Page = "Environment";
let wizardPage: Page = "Environment";
function openPage(next: Page) { page = next; if (next !== "Settings") wizardPage = next; render(); }
let config: Config = { l4t: "", workspace: "", downloadDir: "", logLimit: 5000, proxy: { enabled: false, url: "", rememberCredentials: false } };
let host: Status = { host: false, root: false, checks: [] };
let devices: Device[] = [];
let models: Model[] = [];
let modelID = localStorage.getItem("nvflasher-model") || "";
let flashOptions: Options = { board: "jetson-orin-nano-devkit-super", device: "nvme0n1p1", nvme: "tools/kernel_flash/flash_l4t_t234_nvme.xml", qspi: "bootloader/generic/cfg/flash_t234_qspi.xml", erase: false, showLogs: true };
let provisionOptions: ProvisionOptions = { username: "dev", password: "", hostname: "jetson", publicKey: "", overlay: "", autologin: false, packages: "", script: "", clearKeys: true, clearMachineId: true, firstBoot: true };
let running = false;
let setupStage = "Waiting to download or prepare";
let setupPercent = 0;
let setupFailure = "";
const lines: string[] = [];
let language: Language = localStorage.getItem("nvflasher-language") === "zh-CN" || (!localStorage.getItem("nvflasher-language") && navigator.language.toLowerCase().startsWith("zh")) ? "zh-CN" : "en";
const t = (text: string) => translate(text, language);
let statusText = "Ready";
function status(text: string) { statusText = text; $("#status").textContent = t(text); }
function localizeUI() { localize(document.body, language); status(statusText); document.documentElement.lang = language; }
function log(message: string) { if (document.querySelector<HTMLInputElement>("#diagnostic-log")?.checked) message = `[${new Date().toISOString()}] ${message}`; lines.push(message); if (lines.length > (config.logLimit || 5000)) lines.shift(); const el = $("#log"); el.textContent = lines.join("\n"); if ($<HTMLInputElement>("#autoscroll").checked) el.scrollTop = el.scrollHeight; }
function fail(error: unknown) { const message = error instanceof Error ? error.message : String(error); status("Failed"); log("ERROR: " + message); }
function val(id: string) { return $<HTMLInputElement>("#" + id).value.trim(); }
function check(id: string) { return $<HTMLInputElement>("#" + id).checked; }
function safe(value: string) { return value.replace(/[&<>"']/g, ch => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[ch]!); }
function confirmDanger(message: string) { return Environment.confirm(t("Please confirm"), t(message), t("This operation may change or erase device data. Continue?"), t("Cancel"), t("Continue")); }
async function stream<T = string>(task: (channel: Channel<T>) => Promise<void>, output: (item: T) => void) { if (running) return; running = true; const cleanup = document.querySelector<HTMLButtonElement>("#cleanup-images"); if (cleanup) cleanup.disabled = true; status("Running"); try { await task(new Channel<T>(output)); status("Completed"); log(t("Operation completed")); } catch (error) { fail(error); } finally { running = false; render(); } }
function path(id: string, title: string, current: string, directory = true) { return `<label>${title}<span class="row"><input id="${id}" value="${safe(current)}"/><button type="button" data-path="${id}" data-dir="${directory}">Browse</button></span></label>`; }
function shell(title: string, description: string, body: string) {
  $("#panel").innerHTML = `<h1>${title}</h1><p class="lead">${description}</p>${body}`;
  document.querySelectorAll<HTMLButtonElement>("[data-path]").forEach(button => button.onclick = () => void Environment.browse(button.dataset.dir === "true").then(result => { if (result) { const input = $<HTMLInputElement>("#" + button.dataset.path); input.value = result; input.dispatchEvent(new Event("input", { bubbles: true })); } }).catch(fail));
  const mfi = document.querySelector<HTMLButtonElement>('[data-path="mfi"]');
  if (mfi) { const archive = document.createElement("button"); archive.textContent = "Choose archive"; archive.onclick = () => void Environment.browse(false).then(result => { if (result) $<HTMLInputElement>("#mfi").value = result; }).catch(fail); mfi.after(archive); }
  const advanced = title === "Flash device" ? ["Use a prebuilt MFI package", "Massflash"] : title === "Flash MFI" ? ["Single-device flash", "Flash"] : title === "Devices" ? ["Recovery guide", "Recovery"] : title === "SSH Recovery" ? ["Back to devices", "Devices"] : null;
  if (advanced) {
    const links = document.createElement("div");
    links.className = "page-links";
    const button = document.createElement("button");
    button.textContent = t(advanced[0]!);
    button.onclick = () => openPage(advanced[1] as Page);
    links.append(button);
    $("#panel .lead").after(links);
  }
  if (title === "Environment" && host.host && !host.root) {
    const controls = document.createElement("div");
    controls.className = "sudo-auth";
    controls.innerHTML = `<label>Sudo password<input id="sudo-password" type="password" autocomplete="off"/></label><button id="sudo-restart">Restart with sudo</button><span class="hint">Password is used once for sudo authentication and is not saved.</span>`;
    $("#root").closest(".actions")!.after(controls);
    $("#sudo-restart").onclick = () => {
      const input = $<HTMLInputElement>("#sudo-password");
      const password = input.value;
      input.value = "";
      if (password) void Environment.restartWithSudo(password).catch(fail);
    };
  }
  if (title === "Environment" && host.host) {
    const card = document.createElement("div");
    card.className = "card";
    card.textContent = t("Checking package requirements…");
    $("#report").after(card);
    void Environment.installPlan().then(plan => {
      if (!card.isConnected) return;
      card.replaceChildren();
      const heading = document.createElement("h3");
      heading.textContent = `${t("System dependencies")} · ${plan.distribution}`;
      card.append(heading);
      if (!plan.packages.length && !plan.manualCommands.length) { card.append(t("No missing packages detected.")); return; }
      if (plan.manualCommands.length) {
        const note = document.createElement("p");
        note.className = "warning";
        note.textContent = t("Arch: abootimg is an AUR package, not a pacman repository package. Install it as your regular user (never as root), then rerun diagnostics.");
        const manual = plan.manualCommands.map(args => args.join(" ")).join(" && ");
        const manualCode = document.createElement("pre");
        manualCode.className = "package-command";
        manualCode.textContent = manual;
        const manualCopy = document.createElement("button");
        manualCopy.textContent = t("Copy AUR command");
        manualCopy.onclick = () => void navigator.clipboard.writeText(manual).catch(fail);
        card.append(note, manualCode, manualCopy);
      }
      if (!plan.packages.length) return;
      const summary = document.createElement("p");
      summary.textContent = `${t("Missing packages")}: ${plan.packages.join(", ")}`;
      const commands = plan.commands.map(args => `sudo ${args.join(" ")}`).join(" && ");
      const code = document.createElement("pre");
      code.className = "package-command";
      code.textContent = commands;
      const warning = document.createElement("p");
      warning.className = "hint";
      warning.textContent = plan.commands[0]?.[0] === "pacman" ? t("Arch installation updates the entire system. NVIDIA does not certify Arch as a flashing host.") : t("Package installation changes the host system.");
      const actions = document.createElement("div");
      actions.className = "actions";
      const copy = document.createElement("button");
      copy.textContent = t("Copy install command");
      copy.onclick = () => void navigator.clipboard.writeText(commands).catch(fail);
      const install = document.createElement("button");
      install.className = "primary";
      install.textContent = t("Install missing packages");
      install.disabled = !host.root;
      install.onclick = () => void (async () => { if (await Environment.confirm(t("Please confirm"), t("Install these packages on the host?"), `${commands}\n\n${warning.textContent}`, t("Cancel"), t("Continue"))) await stream(async channel => { await Environment.installDependencies(channel); host = await Environment.capabilities(); }, log); })().catch(fail);
      actions.append(copy, install);
      card.append(summary, code, warning, actions);
    }).catch(fail);
  }
  localize($("#panel"), language);
}
function gate() { return !host.host ? `<div class="warning">Flashing requires a Linux x86_64 host (Linux_for_Tegra, USB, root). Not supported on this platform.</div>` : !host.root ? `<div class="warning">Root required: restart with pkexec or sudo to enable flashing.</div>` : ""; }
function recoveryGuide() {
  return `<section class="card recovery-guide"><h2>${t("Enter Recovery mode and troubleshoot")}</h2>
    <p class="hint">${t("These jumper steps apply to the official Jetson Orin Nano Developer Kit. Check your carrier board manual if its button header differs.")}</p>
    <ol><li>${t("Disconnect the power cable and connect the data USB cable to the host.")}</li>
    <li>${t("On the 12-pin button header, short the pins labeled REC and GND. Do not guess pin positions.")}</li>
    <li>${t("Reconnect power while REC and GND are shorted. After the host detects Recovery USB, remove the jumper.")}</li></ol>
    <h3>${t("Troubleshooting")}</h3>
    <ul><li>${t("No NVIDIA USB device: check power, the data-capable USB cable, and the host USB port; then repeat the power-off sequence.")}</li>
    <li>${t("A running-mode device cannot be flashed: enter Force Recovery and wait for the USB device list to refresh.")}</li>
    <li>${t("For the Orin Nano 8GB module, 0955:7523 (APX) indicates Force Recovery; a stopped fan alone does not identify the mode.")}</li>
    <li>${t("Recovery USB is present but flashing is unavailable: complete BSP preparation on the Download step and rerun Environment diagnostics.")}</li></ul>
    <p class="hint">${t("USB ID alone does not verify the board model. Confirm the hardware label and selected model before flashing.")} <a href="https://docs.nvidia.com/jetson/archives/r36.4.3/DeveloperGuide/IN/QuickStart.html#to-determine-whether-the-developer-kit-is-in-force-recovery-mode" target="_blank" rel="noopener noreferrer">${t("NVIDIA recovery instructions")}</a></p>
  </section>`;
}
function flashForm() { return `<div class="card"><h3>Target configuration</h3><div class="grid"><label>Board preset<select id="preset"><option value="">Custom</option></select></label><label>Board<select id="board"><option value="${safe(flashOptions.board)}">${safe(flashOptions.board)}</option></select></label><label>External device<input id="device" value="${safe(flashOptions.device)}"/></label><label>NVMe layout XML<input id="nvme" value="${safe(flashOptions.nvme)}"/></label><label>QSPI layout XML<input id="qspi" value="${safe(flashOptions.qspi)}"/></label><label class="check"><input id="erase" type="checkbox" ${flashOptions.erase ? "checked" : ""}/> Erase all storage</label><label class="check"><input id="showLogs" type="checkbox" ${flashOptions.showLogs ? "checked" : ""}/> Detailed logs</label></div><p class="hint">rootdev: internal · network: usb0 · Erase all destroys target data.</p></div>`; }
function readFlash() { flashOptions = { board: val("board"), device: val("device"), nvme: val("nvme"), qspi: val("qspi"), erase: check("erase"), showLogs: check("showLogs") }; }
async function fillPresets() { const select = $<HTMLSelectElement>("#preset"); const presets = await Environment.presets(); for (const preset of presets) select.add(new Option(preset.name, preset.name)); select.value = matchingPresetName(presets, flashOptions); select.onchange = () => { const chosen = presets.find(preset => preset.name === select.value); if (chosen) { flashOptions = chosen.options; for (const key of ["board", "device", "nvme", "qspi"] as const) $<HTMLInputElement>("#" + key).value = chosen.options[key]; $<HTMLInputElement>("#erase").checked = chosen.options.erase; $<HTMLInputElement>("#showLogs").checked = chosen.options.showLogs; } }; const report = await Environment.detect(config.l4t); const boardSelect = $<HTMLSelectElement>("#board"); if (!boardSelect.isConnected) return; boardSelect.replaceChildren(); if (!report.boards.includes(flashOptions.board)) { const unavailable = new Option(`${flashOptions.board} — unavailable in this BSP`, flashOptions.board); unavailable.disabled = true; unavailable.selected = true; boardSelect.add(unavailable); } for (const board of report.boards) boardSelect.add(new Option(board, board, false, board === flashOptions.board)); boardSelect.onchange = () => { flashOptions.board = boardSelect.value; select.value = matchingPresetName(presets, flashOptions); }; }
function render() {
  queueMicrotask(localizeUI);
  const nav = $("#nav");
  nav.replaceChildren();
  const current = activeStep(page, wizardPage);
  const sequence: readonly Page[] = steps[workflow];
  const index = sequence.indexOf(current);
  for (const item of ["Flash", "Rootfs"] as const) {
    const button = document.createElement("button");
    button.className = `workflow${workflow === item ? " active" : ""}`;
    button.textContent = t(item === "Flash" ? "Flash setup" : "Rootfs setup");
    button.setAttribute("aria-pressed", String(workflow === item));
    button.onclick = () => { workflow = item; openPage(steps[item][0]); };
    nav.append(button);
    if (workflow === item) {
      const list = document.createElement("div");
      list.className = "wizard-steps";
      for (const [position, step] of steps[item].entries()) {
        const stepButton = document.createElement("button");
        stepButton.textContent = `${position + 1}. ${t(step)}`;
        stepButton.className = step === current && page !== "Settings" ? "active" : "";
        if (stepButton.className) stepButton.setAttribute("aria-current", "step");
        stepButton.onclick = () => openPage(step);
        list.append(stepButton);
      }
      nav.append(list);
    }
  }
  $("#open-settings").onclick = () => openPage("Settings");
  $("#open-settings").classList.toggle("active", page === "Settings");
  $("#wizard-progress").textContent = page === "Settings" ? t("Global settings") : `${t("Step")} ${index + 1} / ${sequence.length}`;
  $("#wizard-title").textContent = page === "Settings" ? t("Settings") : t(workflow === "Flash" ? "Flash setup" : "Rootfs setup");
  const back = $<HTMLButtonElement>("#wizard-back");
  const next = $<HTMLButtonElement>("#wizard-next");
  const position = $("#wizard-position");
  if (page === "Settings") {
    back.disabled = false;
    back.textContent = t("Back to wizard");
    back.onclick = () => openPage(wizardPage);
    next.hidden = true;
    position.textContent = "";
  } else {
    back.textContent = t("Back");
    back.disabled = index <= 0;
    back.onclick = () => openPage(sequence[index - 1]!);
    next.hidden = index < 0 || index >= sequence.length - 1;
    next.disabled = page === "Devices" && !modelID;
    next.textContent = t("Next step");
    next.onclick = () => { if (page !== "Devices" || modelID) openPage(sequence[index + 1]!); };
    position.textContent = `${index + 1} / ${sequence.length}`;
  }
  $("#host").textContent = host.host ? `LINUX x86_64 · ${host.root ? "ROOT READY" : "ROOT REQUIRED"}` : "NON-LINUX FLASHING DISABLED";
  $("#device-count").textContent = `${devices.filter(device => device.mode === "recovery").length} ${t("recovery devices")}`;
  if (page === "Environment") { shell("Environment", "Inspect your Linux for Tegra BSP and host dependencies.", `${gate()}<div class="card">${path("l4t", "Linux_for_Tegra directory", config.l4t)}<div class="actions"><button class="primary" id="detect">Run diagnostics</button><button id="root" ${!host.host || host.root ? "disabled" : ""}>Restart as root</button></div></div><div class="card" id="report">Run diagnostics to inspect BSP.</div>`); $("#detect").onclick = () => void (async () => { try { config.l4t = val("l4t"); await Environment.saveConfig(config); host = await Environment.capabilities(); const report = await Environment.detect(config.l4t); const output = $("#report"); output.replaceChildren(); const heading = document.createElement("h3"); heading.textContent = report.version ? `${report.version} · ${report.boards.length} ${t("boards")}` : t("BSP not prepared — choose a model and download on the previous steps"); output.append(heading); for (const item of [...report.checks, ...host.checks]) { const row = document.createElement("div"); row.className = "report"; const name = document.createElement("span"); name.textContent = t(item.name); const hint = document.createElement("span"); hint.className = item.status; hint.textContent = translateCheck(item.status, item.hint, language); row.append(name, hint); output.append(row); } } catch (error) { fail(error); } })(); $("#root").onclick = () => void Environment.restartAsRoot().catch(fail); if (config.l4t) $<HTMLButtonElement>("#detect").click(); }
  if (page === "Devices") { shell("Devices", "Connected NVIDIA USB devices refresh every two seconds on Linux. Flashing requires Recovery mode.", `${gate()}<div class="card" id="device-list"></div><div class="card"><label>${t("Select device model")}<select id="model"><option value="">${t("Choose model…")}</option>${models.map(model => `<option value="${safe(model.id)}" ${model.id === modelID ? "selected" : ""}>${safe(model.name)}</option>`).join("")}</select></label><p class="hint">${t("USB ID alone does not identify the board. Confirm the model from the device label before flashing.")}</p></div>`); const output = $("#device-list"); output.textContent = devices.length ? "" : t("No NVIDIA USB devices found."); for (const device of devices) { const row = document.createElement("div"); row.className = "report"; row.textContent = `USB ${device.path} · 0955:${device.product} · ${t(device.mode)}${device.boardIds ? ` · ${t("L4T-README board IDs")}: ${device.boardIds}` : ""}${device.release ? ` · L4T ${device.release} (L4T-README)` : ""}`; output.append(row); } $<HTMLSelectElement>("#model").onchange = event => { modelID = (event.target as HTMLSelectElement).value; const chosen = models.find(model => model.id === modelID); if (chosen) flashOptions.board = chosen.board; localStorage.setItem("nvflasher-model", modelID); $<HTMLButtonElement>("#wizard-next").disabled = !modelID; }; }
  if (page === "Flash") {
    const recoveryReady = devices.some(device => device.mode === "recovery");
    shell("Flash device", "Run NVIDIA's official initrd flashing tool for one Recovery device.", `${gate()}${!recoveryReady ? `<div class="warning">${t("No Recovery device detected. Enter Recovery before flashing; a running device cannot be flashed.")}</div>` : ""}<div id="flash-readiness" class="warning">${t("Checking BSP readiness…")}</div><div class="actions"><button id="recovery-help">${t("Recovery guide")}</button><button id="resume-setup" ${!host.root || !modelID || running ? "disabled" : ""}>${t("Continue BSP preparation")}</button></div>${flashForm()}<div class="actions"><button class="primary" id="start" disabled>Start flash</button><button id="cleanup-images" ${running || !host.host || !host.root || !config.l4t ? "disabled" : ""}>${t("Check and clean image resources")}</button><button id="cancel">Cancel</button></div>`);
    const recoveryHelp = document.querySelector<HTMLButtonElement>("#recovery-help");
    if (recoveryHelp) recoveryHelp.onclick = () => openPage("Recovery");
    $("#resume-setup").onclick = () => { workflow = "Flash"; openPage("Download"); $<HTMLButtonElement>("#prepare").click(); };
    void Environment.detect(config.l4t).then(report => {
      const hint = document.querySelector("#flash-readiness");
      const start = document.querySelector<HTMLButtonElement>("#start");
      if (!hint || !start || page !== "Flash") return;
      const missing = report.checks.filter(item => item.status !== "ok");
      const missingAbootimg = host.checks.some(item => item.name === "abootimg" && item.status !== "ok");
      hint.textContent = missing.length ? `${t("BSP preparation incomplete")}: ${missing.map(item => t(item.name)).join(", ")}${running ? ` · ${t("Preparation in progress; wait for completion")}` : ""}` : missingAbootimg ? t("Host is missing abootimg; see Environment for installation instructions.") : `${t("BSP ready")}: ${config.l4t}`;
      $<HTMLButtonElement>("#resume-setup").hidden = missing.length === 0;
      hint.classList.toggle("warning", missing.length > 0 || missingAbootimg);
      hint.classList.toggle("hint", missing.length === 0 && !missingAbootimg);
      start.dataset.bspReady = String(missing.length === 0 && !missingAbootimg);
      start.disabled = running || !host.host || !host.root || !devices.some(device => device.mode === "recovery") || missing.length > 0 || missingAbootimg;
    }).catch(fail);
    void fillPresets().catch(fail);
    $("#start").onclick = () => void (async () => { if (running) return; readFlash(); devices = await Devices.list(); if (!devices.some(device => device.mode === "recovery")) { log("重试需要 Force Recovery 设备：请让 Jetson 重新进入 Recovery，无需重启软件。"); render(); return; } if (!await confirmDanger(flashOptions.erase ? "ERASE ALL enabled. Wipe the connected Jetson?" : "Start flashing the connected Jetson?")) return; log(`[${new Date().toISOString()}] Flash start: ${JSON.stringify(flashOptions)}; USB: ${JSON.stringify(devices)}`); await stream(channel => Flash.start(config.l4t, flashOptions, channel), log); log(`[${new Date().toISOString()}] Flash task ended; retry is available after entering Force Recovery.`); })().catch(fail);
    $("#cleanup-images").onclick = () => void (async () => {
      if (running) return;
      readFlash();
      if (!await Environment.confirm(t("Please confirm"), t("Check and clean image resources"), t("Only idle BSP image mounts will be unmounted and detached. Busy resources are refused. No images are deleted and flashing will not start."), t("Cancel"), t("Continue"))) return;
      await stream(channel => Flash.cleanupImages(config.l4t, channel), log);
    })().catch(fail);
    $("#cancel").onclick = () => void Flash.cancel().catch(fail);
  }
  if (page === "Massflash") {
    shell("Flash MFI", "Flash a prebuilt MFI package or generate a new one from the BSP.", `${gate()}<div class="card"><h3>Prebuilt MFI package</h3><div class="grid">${path("mfi", "MFI directory or .tar.gz archive", config.workspace)}<label>Maximum devices<input id="count" type="number" value="1" min="1" max="64"/></label><label class="check"><input id="mfi-logs" type="checkbox" checked/> Detailed logs</label></div><p class="hint">Only flash devices matching the MFI hardware. Recovery USB is required.</p><div class="actions"><button class="primary" id="bulk" ${!host.host || !host.root || !devices.some(device => device.mode === "recovery") ? "disabled" : ""}>Flash MFI devices</button><button id="cancel">Cancel</button></div></div><details class="card"><summary>Generate new MFI package</summary><p class="hint">Offline generation may require BOARDID. Output: mfi_&lt;board&gt;.tar.gz.</p>${flashForm()}<div class="actions"><button id="start" ${!host.host || !host.root ? "disabled" : ""}>Generate MFI</button></div></details>`);
    void fillPresets().catch(fail);
    $("#bulk").onclick = () => void (async () => { if (!await confirmDanger("Flash all connected matching devices?")) return; await stream(channel => Massflash.flash(val("mfi"), Number(val("count")), check("mfi-logs"), channel), log); })().catch(fail);
    $("#start").onclick = () => void (async () => { readFlash(); if (!await confirmDanger(flashOptions.erase ? "ERASE ALL enabled. Wipe the connected Jetson?" : "Generate a new MFI package?")) return; await stream(channel => Massflash.generate(config.l4t, flashOptions, Number(val("count")), channel), log); })().catch(fail);
    $("#cancel").onclick = () => void Massflash.cancel().catch(fail);
  }
  if (page === "Provision") {
    shell("Provision rootfs", "Create a user, install packages and customize rootfs using a non-interactive chroot.", `${gate()}<div class="warning">Scripts run as root with no TTY. Review the entire script before running it.</div><div class="card">${path("provision-l4t", "Linux_for_Tegra directory", config.l4t)}<p class="hint">${t("Rootfs to configure")}: ${config.l4t ? safe(config.l4t + "/rootfs") : t("No prepared rootfs selected")}</p><button id="prepare-rootfs">${t("Prepare rootfs in Flash wizard")}</button></div><div class="card grid"><label>Username<input id="username" value="${safe(provisionOptions.username)}"/></label><label>Hostname prefix<input id="hostname" value="${safe(provisionOptions.hostname)}"/></label><label>Password<span class="row"><input id="password" type="password" value="${safe(provisionOptions.password)}"/><button id="show-password">Show</button><button id="random">Generate</button></span></label>${path("publicKey", "SSH public key", provisionOptions.publicKey, false)}${path("overlay", "Overlay directory", provisionOptions.overlay)}<label class="check"><input id="autologin" type="checkbox" ${provisionOptions.autologin ? "checked" : ""}/> Autologin</label></div><div class="card grid"><label class="wide">APT packages (space/newline separated)<textarea id="packages">${safe(provisionOptions.packages)}</textarea></label><label class="wide">Chroot script (stdin, non-interactive)<textarea id="script">${safe(provisionOptions.script)}</textarea></label><div class="wide"><h3>${t("Script templates")}</h3><p class="hint">${t("Select multiple templates, then add their scripts above. Only the visible script runs when you provision.")}</p><div id="templates" class="template-list"></div><p id="template-selection" class="hint"></p><div class="actions"><button id="load-template">${t("Add selected scripts")}</button><button id="save-template">${t("Save as")}</button><button id="rename-template">${t("Rename")}</button><button id="delete-template">${t("Delete")}</button></div></div><label class="check"><input id="clearKeys" type="checkbox" checked/> Clear SSH host keys</label><label class="check"><input id="clearMachineId" type="checkbox" checked/> Clear machine-id</label><label class="check"><input id="firstBoot" type="checkbox" checked/> First-boot identity service</label></div><div class="actions"><button class="primary" id="run" ${!host.host || !host.root || !config.l4t ? "disabled" : ""}>Run provisioning</button><button id="cancel">Cancel</button></div>`);
    $("#prepare-rootfs").onclick = () => { workflow = "Flash"; openPage("Download"); };
    $<HTMLInputElement>("#provision-l4t").oninput = () => { $<HTMLButtonElement>("#run").disabled = !host.root || !host.host || !val("provision-l4t"); };
    void setupProvision();
  }
  if (page === "Recovery") { shell("SSH Recovery", "Reboot a running Jetson into forced recovery over SSH. Available on all platforms.", `${recoveryGuide()}<details class="card"><summary>${t("Enter Recovery via SSH")}</summary><p class="hint">${t("Requires a bootable device, SSH access and a trusted host key in ~/.ssh/known_hosts. Otherwise use FC REC + GND or RECOVERY + RESET.")}</p><div class="grid"><label>Host/IP<input id="ssh-host"/></label><label>Port<input id="port" value="22"/></label><label>Username<input id="user"/></label><label>SSH password<input id="ssh-pass" type="password"/></label><label>Sudo password (optional)<input id="sudo-pass" type="password"/></label></div><button class="primary" id="reboot">Request forced recovery</button></details>`); $("#reboot").onclick = () => void (async () => { if (!await confirmDanger("Reboot the device into Recovery mode?")) return; await Recovery.trigger({ host: val("ssh-host"), port: val("port"), user: val("user"), password: val("ssh-pass"), sudoPassword: val("sudo-pass") }); log("Waiting for USB Recovery 0955:7X23"); openPage("Devices"); })().catch(fail); }
  if (page === "Download") {
    const model = models.find(item => item.id === modelID);
    shell("Download images", "Download the official BSP (including flash tools) and rootfs for the selected model.", `<div class="card"><h3>${model ? safe(model.name) : t("Choose a model on the Devices step first")}</h3>${model ? `<p class="hint">Jetson Linux ${safe(model.release)} · ${model.packages.map(pkg => safe(pkg.name)).join(" · ")}</p>` : ""}<div class="grid">${path("download-dir", "Save directory", config.downloadDir)}<label>${t("Proxy override (HTTP/HTTPS/SOCKS5)")}<input id="download-proxy" value="${safe(config.proxy.enabled ? config.proxy.url : "")}" placeholder="socks5://127.0.0.1:10808"/></label></div><p class="hint">${t("Archives are verified with NVIDIA SHA1 hashes before extraction. Preparation requires root and can use substantial disk space.")}</p></div><div class="actions"><button class="primary" id="download" ${!model || !host.host ? "disabled" : ""}>${t("Download BSP, flash tools and rootfs")}</button><button id="prepare" ${!model || !host.root ? "disabled" : ""}>${t("Extract and prepare BSP")}</button><button id="cancel">${t("Cancel")}</button></div><div class="card" role="status" aria-live="polite"><p id="setup-stage">${safe(t(setupStage))}</p><progress id="setup-progress" value="${setupPercent}" max="100"></progress>${setupFailure ? `<p class="warning">${safe(setupFailure)}</p>` : ""}</div>`);
    const showProgress = (line: string) => { log(line); setupStage = line; const current = document.querySelector("#setup-stage"); if (current) current.textContent = t(line); const progress = document.querySelector<HTMLProgressElement>("#setup-progress"); const match = line.match(/: (\d+)%$/); if (match) setupPercent = Number(match[1]); else setupPercent = 0; if (progress) progress.value = setupPercent; };
    const prepare = async (selectedModel: string, directory: string) => {
      config.l4t = await Setup.location(selectedModel, directory);
      await Environment.saveConfig(config);
      try { await Setup.prepare(selectedModel, directory, new Channel<string>(showProgress)); }
      catch (error) { setupFailure = error instanceof Error ? error.message : String(error); throw error; }
      setupFailure = "";
      log(`${t("BSP ready")}: ${config.l4t}`);
    };
    $("#download").onclick = () => { const selectedModel = modelID, directory = val("download-dir"), proxy = val("download-proxy"); config.downloadDir = directory; void stream<string>(async channel => { await Environment.saveConfig(config); await Setup.download(selectedModel, directory, proxy, channel); if (host.root) await prepare(selectedModel, directory); else log(t("Download complete. Restart with sudo, then select Extract and prepare BSP.")); }, showProgress); };
    $("#prepare").onclick = () => { const selectedModel = modelID, directory = val("download-dir"); config.downloadDir = directory; void stream<string>(async () => { await prepare(selectedModel, directory); }, showProgress); };
    $("#cancel").onclick = () => void Setup.cancel().catch(fail);
  }
  if (page === "Settings") { shell("Settings", "Saved locally. SSH, sudo and device passwords are never stored.", `<div class="card grid">${path("settings-l4t", "Linux_for_Tegra", config.l4t)}${path("workspace", "Workspace", config.workspace)}${path("settings-download", "Download directory", config.downloadDir)}<label>Maximum log lines<input id="log-limit" type="number" min="100" value="${config.logLimit}"/></label><label class="check"><input id="proxy-enabled" type="checkbox" ${config.proxy.enabled ? "checked" : ""}/> Enable proxy</label><label>HTTP/HTTPS/SOCKS5 proxy URL<input id="proxy-url" value="${safe(config.proxy.url)}"/></label><label class="check wide"><input id="remember" type="checkbox" ${config.proxy.rememberCredentials ? "checked" : ""}/> Remember proxy credentials (stored on disk; security risk)</label></div><div class="actions"><button class="primary" id="save-settings">Save settings</button><button id="test-proxy">Test proxy</button></div>`); $("#save-settings").onclick = () => { config = { l4t: val("settings-l4t"), workspace: val("workspace"), downloadDir: val("settings-download"), logLimit: Number(val("log-limit")), proxy: { enabled: check("proxy-enabled"), url: val("proxy-url"), rememberCredentials: check("remember") } }; void Environment.saveConfig(config).then(() => { $("#status").textContent = "Settings saved"; }).catch(fail); }; $("#test-proxy").onclick = () => void Download.testProxy(val("proxy-url")).then(() => { $("#status").textContent = "Proxy reachable"; }).catch(fail); }
}
async function setupProvision() {
  const password = $<HTMLInputElement>("#password");
  $("#show-password").onclick = () => { password.type = password.type === "password" ? "text" : "password"; };
  $("#random").onclick = () => void Provision.generatePassword().then(value => { password.value = value; }).catch(fail);
  const selected = new Set<string>();
  let availableTemplates: Template[] = [];
  const choices = $("#templates");
  const selection = $("#template-selection");
  const nameLabel = document.createElement("label");
  nameLabel.textContent = t("Template name");
  const nameInput = document.createElement("input");
  nameInput.id = "template-name";
  nameLabel.append(nameInput);
  selection.after(nameLabel);
  const refresh = async () => {
    availableTemplates = await Provision.listTemplates();
    if (!choices.isConnected) return;
    choices.replaceChildren();
    for (const template of availableTemplates) {
      const label = document.createElement("label");
      label.className = "check";
      const box = document.createElement("input");
      box.type = "checkbox";
      box.checked = selected.has(template.name);
      box.onchange = () => { if (box.checked) selected.add(template.name); else selected.delete(template.name); updateSelection(); };
      label.append(box, document.createTextNode(` ${template.name}${template.builtIn ? ` · ${t("built-in")}` : ""}`));
      choices.append(label);
    }
    updateSelection();
  };
  const updateSelection = () => {
    selection.textContent = selected.size ? `${t("Selected templates")}: ${[...selected].join(", ")}` : t("No templates selected");
    $<HTMLButtonElement>("#load-template").disabled = !selected.size;
    const single = [...selected].length === 1 ? [...selected][0] : "";
    const readOnly = availableTemplates.find(template => template.name === single)?.builtIn;
    $<HTMLButtonElement>("#rename-template").disabled = !single || !!readOnly;
    $<HTMLButtonElement>("#delete-template").disabled = !single || !!readOnly;
  };
  void refresh().catch(fail);
  $("#load-template").onclick = () => void (async () => {
    const templates = await Promise.all([...selected].map(name => Provision.loadTemplate(name)));
    const script = $<HTMLTextAreaElement>("#script");
    script.value = appendTemplateScripts(script.value, templates.map(template => template.script));
    provisionOptions.script = script.value;
    selection.textContent = `${t("Scripts added to the editor")}: ${templates.map(template => template.name).join(", ")}`;
    script.focus();
  })().catch(fail);
  $("#save-template").onclick = () => { const name = nameInput.value.trim(); if (!name) { nameInput.focus(); return; } void Provision.saveTemplate(name, val("script")).then(refresh).catch(fail); };
  $("#rename-template").onclick = () => { const name = nameInput.value.trim(); const oldName = [...selected][0]; if (!name) { nameInput.focus(); return; } if (oldName) void Provision.renameTemplate(oldName, name).then(() => { selected.delete(oldName); selected.add(name); return refresh(); }).catch(fail); };
  $("#delete-template").onclick = () => void (async () => { const name = [...selected][0]; if (!name || !await Environment.confirm(t("Please confirm"), t("Delete selected template?"), name, t("Cancel"), t("Continue"))) return; await Provision.deleteTemplate(name); selected.delete(name); await refresh(); })().catch(fail);
  $("#run").onclick = () => void (async () => { config.l4t = val("provision-l4t"); await Environment.saveConfig(config); provisionOptions = { username: val("username"), hostname: val("hostname"), password: password.value, publicKey: val("publicKey"), overlay: val("overlay"), autologin: check("autologin"), packages: val("packages"), script: $<HTMLTextAreaElement>("#script").value, clearKeys: check("clearKeys"), clearMachineId: check("clearMachineId"), firstBoot: check("firstBoot") }; if (!await Environment.confirm(t("Please confirm"), t("Run rootfs provisioning?"), `${t("Full chroot script:")}\n${provisionOptions.script || t("(none)")}`, t("Cancel"), t("Continue"))) return; await stream(channel => Provision.run(config.l4t, provisionOptions, channel), log); })().catch(fail);
  $("#cancel").onclick = () => void Provision.cancel().catch(fail);
}
const diagnosticLabel = document.createElement("label");
diagnosticLabel.className = "check";
diagnosticLabel.innerHTML = '<input id="diagnostic-log" type="checkbox" checked/> 写入诊断日志（软件目录/logs）';
$("#copy").parentElement?.append(diagnosticLabel);
const diagnosticToggle = $<HTMLInputElement>("#diagnostic-log");
diagnosticToggle.checked = localStorage.getItem("nvflasher-diagnostic-log") !== "false";
diagnosticToggle.onchange = () => { localStorage.setItem("nvflasher-diagnostic-log", String(diagnosticToggle.checked)); void Environment.setLogging(diagnosticToggle.checked).catch(fail); };
await Environment.setLogging(diagnosticToggle.checked);
$("#copy").onclick = () => void navigator.clipboard.writeText(lines.join("\n")).catch(fail);
$("#save").onclick = () => { const link = document.createElement("a"); link.href = URL.createObjectURL(new Blob([lines.join("\n")], { type: "text/plain" })); link.download = "nvflasher.log"; link.click(); setTimeout(() => URL.revokeObjectURL(link.href), 1000); };
$("#hide").onclick = () => { $("#drawer").classList.toggle("hidden"); $("#hide").textContent = t($("#drawer").classList.contains("hidden") ? "Show" : "Hide"); };
events.devicesChanged.on(next => { const changed = JSON.stringify(devices) !== JSON.stringify(next); devices = next; $("#device-count").textContent = `${devices.filter(device => device.mode === "recovery").length} ${t("recovery devices")}`; if (changed) log(`[${new Date().toISOString()}] USB changed: ${JSON.stringify(next)}`); if (page === "Flash") { const start = document.querySelector<HTMLButtonElement>("#start"); if (start) start.disabled = running || !host.host || !host.root || start.dataset.bspReady !== "true" || !devices.some(device => device.mode === "recovery"); } else if (changed && !running && (page === "Devices" || page === "Massflash")) render(); });
const languageSelect = $<HTMLSelectElement>("#language");
languageSelect.value = language;
languageSelect.onchange = () => {
  language = languageSelect.value as Language;
  localStorage.setItem("nvflasher-language", language);
  localizeUI();
  $("#device-count").textContent = `${devices.filter(device => device.mode === "recovery").length} ${t("recovery devices")}`;
  $("#hide").textContent = t($("#drawer").classList.contains("hidden") ? "Show" : "Hide");
  document.querySelectorAll<HTMLButtonElement>("#nav .workflow").forEach((button, index) => { button.textContent = t(index === 0 ? "Flash setup" : "Rootfs setup"); });
  document.querySelectorAll<HTMLButtonElement>("#nav .wizard-steps button").forEach((button, index) => { button.textContent = `${index + 1}. ${t(steps[workflow][index]!)}`; });
  const current = activeStep(page, wizardPage);
  const sequence: readonly Page[] = steps[workflow];
  $("#wizard-progress").textContent = page === "Settings" ? t("Global settings") : `${t("Step")} ${sequence.indexOf(current) + 1} / ${sequence.length}`;
  $("#wizard-title").textContent = t(page === "Settings" ? "Settings" : workflow === "Flash" ? "Flash setup" : "Rootfs setup");
  $("#wizard-back").textContent = t(page === "Settings" ? "Back to wizard" : "Back");
  $("#wizard-next").textContent = t("Next step");
  const advanced = document.querySelector<HTMLButtonElement>(".page-links button");
  if (advanced) advanced.textContent = t(page === "Massflash" ? "Single-device flash" : page === "Flash" ? "Use a prebuilt MFI package" : page === "Recovery" ? "Back to devices" : "SSH Recovery");
};
$("#panel").addEventListener("change", () => {
  if ((page === "Flash" || page === "Massflash") && document.querySelector("#board")) { readFlash(); config.flash = flashOptions; void Environment.saveConfig(config).catch(fail); }
  if (page === "Provision" && document.querySelector("#username")) {
    provisionOptions = { username: val("username"), hostname: val("hostname"), password: $<HTMLInputElement>("#password").value, publicKey: val("publicKey"), overlay: val("overlay"), autologin: check("autologin"), packages: val("packages"), script: $<HTMLTextAreaElement>("#script").value, clearKeys: check("clearKeys"), clearMachineId: check("clearMachineId"), firstBoot: check("firstBoot") };
    const { password: _password, script: _script, ...saved } = provisionOptions;
    config.provision = { ...saved, script: "" };
    void Environment.saveConfig(config).catch(fail);
  }
});
try {
  config = await Environment.getConfig();
  if (config.flash?.board) flashOptions = config.flash;
  if (config.provision) provisionOptions = { ...provisionOptions, ...config.provision };
  host = await Environment.capabilities();
  provisionOptions.password = await Provision.generatePassword();
  models = await Setup.models();
  const chosen = models.find(model => model.id === modelID);
  if (chosen) {
    flashOptions.board = chosen.board;
    if (!config.l4t && config.downloadDir) {
      const candidate = await Setup.location(chosen.id, config.downloadDir);
      if ((await Environment.detect(candidate)).checks.some(check => check.status === "ok")) config.l4t = candidate;
    }
  }
  devices = await Devices.list();
  render();
  if (host.host) void Devices.startWatch().catch(fail);
  if (!sessionStorage.getItem("eula-notice")) { sessionStorage.setItem("eula-notice", "1"); log("NVIDIA BSP is subject to NVIDIA EULA. Flashing may erase data. Proceed at your own risk."); }
} catch (error) { fail(error); }
