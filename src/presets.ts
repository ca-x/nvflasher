import type { Options, Preset } from "./mygo";

export function matchingPresetName(presets: Preset[], options: Options): string {
  return presets.find(preset => preset.options.board === options.board
    && preset.options.device === options.device
    && preset.options.nvme === options.nvme
    && preset.options.qspi === options.qspi)?.name ?? "";
}
