export function appendDebPath(current: string, selected: string): string {
  const path = selected.trim();
  if (!path || current.split("\n").some(line => line.trim() === path)) return current;
  return current.trimEnd() + (current.trim() ? "\n" : "") + path;
}

export function appendDebSelection(input: { value: string; dispatchEvent(event: Event): boolean }, selected: string): void {
  const next = appendDebPath(input.value, selected);
  if (next === input.value) return;
  input.value = next;
  input.dispatchEvent(new Event("input", { bubbles: true }));
  input.dispatchEvent(new Event("change", { bubbles: true }));
}
