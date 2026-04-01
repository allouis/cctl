import { describe, it, expect } from "vitest";
import { pairedResultIds, findToolResult, shortenToolText } from "./transcript";
import type { TranscriptEntry } from "../types";

function entry(
  role: TranscriptEntry["role"],
  text: string,
  opts?: { tool_use_id?: string; is_error?: boolean },
): TranscriptEntry {
  return { role, text, ...opts };
}

describe("pairedResultIds", () => {
  it("pairs tool_use with its following tool_result", () => {
    const entries = [
      entry("tool_use", "Read foo.ts", { tool_use_id: "t1" }),
      entry("tool_result", "contents", { tool_use_id: "t1" }),
    ];
    expect(pairedResultIds(entries)).toEqual(new Set(["t1"]));
  });

  it("handles multiple pairs", () => {
    const entries = [
      entry("tool_use", "Read a", { tool_use_id: "t1" }),
      entry("tool_result", "a contents", { tool_use_id: "t1" }),
      entry("tool_use", "Read b", { tool_use_id: "t2" }),
      entry("tool_result", "b contents", { tool_use_id: "t2" }),
    ];
    expect(pairedResultIds(entries)).toEqual(new Set(["t1", "t2"]));
  });

  it("does not pair if result comes before use", () => {
    const entries = [
      entry("tool_result", "orphan", { tool_use_id: "t1" }),
      entry("tool_use", "Read a", { tool_use_id: "t1" }),
    ];
    // The result at index 0 comes before the use at index 1,
    // but pairing searches forward from the use — the result at 0
    // won't be found by looking forward from index 1
    expect(pairedResultIds(entries)).toEqual(new Set());
  });

  it("returns empty set for no tool entries", () => {
    const entries = [
      entry("user", "hello"),
      entry("assistant", "hi"),
    ];
    expect(pairedResultIds(entries)).toEqual(new Set());
  });

  it("handles tool_use without matching result", () => {
    const entries = [
      entry("tool_use", "Read a", { tool_use_id: "t1" }),
      entry("assistant", "done"),
    ];
    expect(pairedResultIds(entries)).toEqual(new Set());
  });

  it("handles entries with interleaved assistant messages", () => {
    const entries = [
      entry("tool_use", "Read a", { tool_use_id: "t1" }),
      entry("assistant", "thinking..."),
      entry("tool_result", "a contents", { tool_use_id: "t1" }),
    ];
    expect(pairedResultIds(entries)).toEqual(new Set(["t1"]));
  });
});

describe("findToolResult", () => {
  it("finds matching result after the tool_use", () => {
    const entries = [
      entry("tool_use", "Read foo", { tool_use_id: "t1" }),
      entry("tool_result", "contents", { tool_use_id: "t1" }),
    ];
    expect(findToolResult(entries, 0)).toEqual(entries[1]);
  });

  it("returns null when no matching result exists", () => {
    const entries = [
      entry("tool_use", "Read foo", { tool_use_id: "t1" }),
      entry("assistant", "done"),
    ];
    expect(findToolResult(entries, 0)).toBeNull();
  });

  it("returns null for entry without tool_use_id", () => {
    const entries = [
      entry("tool_use", "Read foo"),
    ];
    expect(findToolResult(entries, 0)).toBeNull();
  });

  it("returns null for out of bounds index", () => {
    expect(findToolResult([], 5)).toBeNull();
  });

  it("skips results with different tool_use_id", () => {
    const entries = [
      entry("tool_use", "Read a", { tool_use_id: "t1" }),
      entry("tool_result", "b contents", { tool_use_id: "t2" }),
      entry("tool_result", "a contents", { tool_use_id: "t1" }),
    ];
    expect(findToolResult(entries, 0)).toEqual(entries[2]);
  });

  it("preserves is_error on returned result", () => {
    const entries = [
      entry("tool_use", "Bash cmd", { tool_use_id: "t1" }),
      entry("tool_result", "error output", { tool_use_id: "t1", is_error: true }),
    ];
    const result = findToolResult(entries, 0);
    expect(result?.is_error).toBe(true);
  });
});

describe("shortenToolText", () => {
  it("strips cwd prefix from paths", () => {
    expect(shortenToolText("Read /Users/egg/Code/project/src/main.ts", "/Users/egg/Code/project"))
      .toBe("Read src/main.ts");
  });

  it("replaces bare cwd with dot", () => {
    expect(shortenToolText("Glob /Users/egg/Code/project", "/Users/egg/Code/project"))
      .toBe("Glob .");
  });

  it("handles empty cwd", () => {
    expect(shortenToolText("Read /full/path.ts", "")).toBe("Read /full/path.ts");
  });

  it("handles multiple path occurrences", () => {
    expect(shortenToolText("$ cd /tmp/proj && ls /tmp/proj/src", "/tmp/proj"))
      .toBe("$ cd . && ls src");
  });
});
