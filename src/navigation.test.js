import { expect, test } from "bun:test";
import { activeStep, steps } from "./navigation.ts";

test("flash and rootfs remain separate wizard paths", () => {
  expect(steps.Flash).toEqual(["Environment", "Devices", "Download", "Flash"]);
  expect(steps.Rootfs).toEqual(["Environment", "Provision"]);
});

test("advanced screens keep the right wizard step", () => {
  expect(activeStep("Massflash", "Environment")).toBe("Flash");
  expect(activeStep("Recovery", "Environment")).toBe("Devices");
  expect(activeStep("Settings", "Massflash")).toBe("Flash");
});
