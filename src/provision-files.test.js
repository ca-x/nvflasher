import { expect, test } from "bun:test";
import { appendDebPath, appendDebSelection } from "./provision-files.ts";

test("DEB selection appends without duplicates or changing existing paths", () => {
  expect(appendDebPath("", "/tmp/a.deb")).toBe("/tmp/a.deb");
  expect(appendDebPath("/tmp/a.deb\n", "/tmp/b.deb")).toBe("/tmp/a.deb\n/tmp/b.deb");
  expect(appendDebPath(" /tmp/a.deb \n", "/tmp/a.deb")).toBe(" /tmp/a.deb \n");
  expect(appendDebPath("/tmp/a.deb", "")).toBe("/tmp/a.deb");
});

test("DEB browse dispatches persistence events only when selection changes", () => {
  const events = [];
  const input = { value: "", dispatchEvent(event) { events.push([event.type, event.bubbles]); return true; } };
  appendDebSelection(input, "/tmp/a.deb");
  expect(input.value).toBe("/tmp/a.deb");
  expect(events).toEqual([["input", true], ["change", true]]);
  appendDebSelection(input, "/tmp/a.deb");
  appendDebSelection(input, "");
  expect(events.length).toBe(2);
});
