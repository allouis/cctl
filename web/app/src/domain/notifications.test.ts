import { describe, it, expect } from "vitest";
import {
  detectTransitions,
  buildStateMap,
  type SessionSnapshot,
} from "./notifications";

function snap(id: string, name: string, executor_state: string, preview = ""): SessionSnapshot {
  return { session_id: id, name, executor_state, preview };
}

describe("detectTransitions", () => {
  it("detects NEEDS_INPUT transition", () => {
    const prev = new Map([["s1", "WORKING"]]);
    const current = [snap("s1", "alpha", "NEEDS_INPUT", "Allow read?")];
    const result = detectTransitions(prev, current);
    expect(result).toEqual([{ name: "alpha", kind: "needs_input", executor_detail: "Allow read?" }]);
  });

  it("detects WORKING → IDLE (finished)", () => {
    const prev = new Map([["s1", "WORKING"]]);
    const current = [snap("s1", "alpha", "IDLE", "Done with task")];
    const result = detectTransitions(prev, current);
    expect(result).toEqual([{ name: "alpha", kind: "finished", executor_detail: "Done with task" }]);
  });

  it("detects WORKING → DONE (ended)", () => {
    const prev = new Map([["s1", "WORKING"]]);
    const current = [snap("s1", "alpha", "DONE")];
    const result = detectTransitions(prev, current);
    expect(result).toEqual([{ name: "alpha", kind: "ended", executor_detail: "Session ended" }]);
  });

  it("ignores session that was already NEEDS_INPUT", () => {
    const prev = new Map([["s1", "NEEDS_INPUT"]]);
    const current = [snap("s1", "alpha", "NEEDS_INPUT", "Allow read?")];
    expect(detectTransitions(prev, current)).toEqual([]);
  });

  it("ignores IDLE → IDLE (no change)", () => {
    const prev = new Map([["s1", "IDLE"]]);
    const current = [snap("s1", "alpha", "IDLE")];
    expect(detectTransitions(prev, current)).toEqual([]);
  });

  it("ignores new sessions (no previous state)", () => {
    const prev = new Map<string, string>();
    const current = [snap("s1", "alpha", "WORKING")];
    expect(detectTransitions(prev, current)).toEqual([]);
  });

  it("detects multiple transitions", () => {
    const prev = new Map([
      ["s1", "WORKING"],
      ["s2", "WORKING"],
    ]);
    const current = [
      snap("s1", "alpha", "IDLE", "done A"),
      snap("s2", "beta", "NEEDS_INPUT", "perm B"),
    ];
    const result = detectTransitions(prev, current);
    expect(result).toHaveLength(2);
    expect(result[0]!.kind).toBe("finished");
    expect(result[1]!.kind).toBe("needs_input");
  });
});

describe("buildStateMap", () => {
  it("builds map keyed by session_id", () => {
    const sessions = [snap("s1", "a", "WORKING"), snap("s2", "b", "NEEDS_INPUT")];
    const map = buildStateMap(sessions);
    expect(map.get("s1")).toBe("WORKING");
    expect(map.get("s2")).toBe("NEEDS_INPUT");
    expect(map.size).toBe(2);
  });

  it("returns empty map for empty input", () => {
    expect(buildStateMap([]).size).toBe(0);
  });
});
