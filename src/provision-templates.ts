export function appendTemplateScripts(current: string, templates: string[]): string {
  return [current.trim(), ...templates.map(script => script.trim())].filter(Boolean).join("\n\n");
}
