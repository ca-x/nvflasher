import { expect, test } from "bun:test";
import { matchingPresetName } from "./presets.ts";

test("selected model shows its matching board preset", () => {
  const basic = { board: "jetson-orin-nano-devkit", device: "nvme0n1p1", nvme: "nvme.xml", qspi: "qspi.xml", erase: false, showLogs: true };
  const superModel = { ...basic, board: "jetson-orin-nano-devkit-super" };
  const presets = [{ name: "Nano", options: basic }, { name: "Super", options: superModel }];
  expect(matchingPresetName(presets, superModel)).toBe("Super");
  expect(matchingPresetName(presets, basic)).toBe("Nano");
  expect(matchingPresetName(presets, { ...superModel, nvme: "custom.xml" })).toBe("");
});
