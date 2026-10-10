import { expect, test } from "bun:test";
import { translate, translateCheck } from "./i18n.ts";

test("translates known UI messages and preserves technical strings", () => {
  expect(translate("Restart with sudo", "zh-CN")).toBe("输入密码并提权重启");
  expect(translate("Erase all storage", "zh-CN")).toBe("擦除全部存储");
  expect(translate("Erase all storage", "en")).toBe("Erase all storage");
  expect(translate("0955:7023", "zh-CN")).toBe("0955:7023");
});

test("incomplete BSP points to the preparation action", () => {
  expect(translateCheck("error", "Missing rootfs/etc/nv_tegra_release; run BSP preparation in the Download step", "zh-CN"))
    .toContain("解压并准备 BSP");
});

test("recovery instructions and troubleshooting are available in Chinese", () => {
  expect(translate("Enter Recovery mode and troubleshoot", "zh-CN")).toContain("短接");
  expect(translate("On the 12-pin button header, short the pins labeled REC and GND. Do not guess pin positions.", "zh-CN")).toContain("REC 和 GND");
  expect(translate("For the Orin Nano 8GB module, 0955:7523 (APX) indicates Force Recovery; a stopped fan alone does not identify the mode.", "zh-CN")).toContain("0955:7523");
});

test("Arch AUR instructions name the required executable", () => {
  expect(translate("Arch: abootimg is an AUR package, not a pacman repository package. Install it as your regular user (never as root), then rerun diagnostics.", "zh-CN"))
    .toContain("abootimg");
  expect(translate("Host is missing abootimg; see Environment for installation instructions.", "zh-CN"))
    .toContain("abootimg");
});
