export const steps = {
  Flash: ["Environment", "Devices", "Download", "Flash"],
  Rootfs: ["Environment", "Provision"],
} as const;

export type Workflow = keyof typeof steps;
export type Page = typeof steps.Flash[number] | typeof steps.Rootfs[number] | "Massflash" | "Recovery" | "Settings";

export function activeStep(page: Page, previous: Page): Page {
  if (page === "Settings") return activeStep(previous, "Environment");
  if (page === "Massflash") return "Flash";
  if (page === "Recovery") return "Devices";
  return page;
}
