import { expect, test } from "bun:test";
import { appendTemplateScripts } from "./provision-templates.ts";

test("combines selected scripts while keeping edits visible", () => {
  expect(appendTemplateScripts("echo existing", ["echo first\n", "echo second"])).toBe("echo existing\n\necho first\n\necho second");
  expect(appendTemplateScripts("", [])).toBe("");
});
