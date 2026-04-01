import type { TranscriptEntry } from "../types";

/**
 * Build a set of tool_use_ids where the tool_result has been paired
 * inline with its tool_use (i.e., the result appears after the use
 * in the entries array). These results should not be rendered standalone.
 */
export function pairedResultIds(entries: readonly TranscriptEntry[]): Set<string> {
  const set = new Set<string>();
  for (let i = 0; i < entries.length; i++) {
    const e = entries[i]!;
    if (e.role === "tool_use" && e.tool_use_id) {
      for (let j = i + 1; j < entries.length; j++) {
        const r = entries[j]!;
        if (r.role === "tool_result" && r.tool_use_id === e.tool_use_id) {
          set.add(r.tool_use_id);
          break;
        }
      }
    }
  }
  return set;
}

/**
 * Find the tool_result entry that matches a given tool_use entry.
 * Searches forward from the tool_use's position in the array.
 */
export function findToolResult(
  entries: readonly TranscriptEntry[],
  toolUseIndex: number,
): TranscriptEntry | null {
  const entry = entries[toolUseIndex];
  if (!entry || !entry.tool_use_id) return null;

  for (let j = toolUseIndex + 1; j < entries.length; j++) {
    const r = entries[j]!;
    if (r.role === "tool_result" && r.tool_use_id === entry.tool_use_id) {
      return r;
    }
  }
  return null;
}

export function shortenToolText(text: string, cwd: string): string {
  if (!cwd || !text) return text;
  return text.split(cwd + "/").join("").split(cwd).join(".");
}
