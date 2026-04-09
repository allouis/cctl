import { describe, it, expect } from "vitest";
import { fuzzyScore, searchRepos, getRecentRepos } from "./repos";

const repos = [
  "/home/agent/ghost-cloud-native",
  "/home/agent/cctl",
  "/home/agent/nix",
  "/home/agent/llm-agents",
  "/home/agent/scheduler",
  "/home/agent/scam-detector",
];

describe("fuzzyScore", () => {
  it("returns 100 for exact basename match", () => {
    expect(fuzzyScore("/home/agent/cctl", "cctl")).toBe(100);
  });

  it("returns 80 for basename prefix", () => {
    expect(fuzzyScore("/home/agent/ghost-cloud-native", "ghost")).toBe(80);
  });

  it("returns 60 for basename substring", () => {
    expect(fuzzyScore("/home/agent/ghost-cloud-native", "cloud")).toBe(60);
  });

  it("returns 40 for full path substring", () => {
    expect(fuzzyScore("/home/agent/cctl", "agent/cc")).toBe(40);
  });

  it("returns 20 for fuzzy character match on basename", () => {
    expect(fuzzyScore("/home/agent/scheduler", "shdr")).toBe(20);
  });

  it("returns 0 for no match", () => {
    expect(fuzzyScore("/home/agent/cctl", "xyz")).toBe(0);
  });

  it("returns 1 for empty query (everything matches)", () => {
    expect(fuzzyScore("/home/agent/cctl", "")).toBe(1);
  });
});

describe("searchRepos", () => {
  it("returns top results sorted by score", () => {
    const results = searchRepos(repos, "sc", 5);
    // Both start with "sc" so both score 80; order within same score is stable
    expect(results.map((r) => r.basename)).toContain("scam-detector");
    expect(results.map((r) => r.basename)).toContain("scheduler");
    // A non-prefix match should rank lower
    const results2 = searchRepos(repos, "cctl", 5);
    expect(results2[0]!.basename).toBe("cctl");
  });

  it("respects limit", () => {
    const results = searchRepos(repos, "", 3);
    expect(results).toHaveLength(3);
  });

  it("filters out non-matches", () => {
    const results = searchRepos(repos, "zzzzz", 10);
    expect(results).toHaveLength(0);
  });

  it("ranks exact match above prefix", () => {
    const results = searchRepos(
      ["/a/nix", "/a/nix-config", "/a/nixos"],
      "nix",
      5,
    );
    expect(results[0]!.basename).toBe("nix");
  });
});

describe("getRecentRepos", () => {
  it("filters out repos no longer in the list", () => {
    const result = getRecentRepos(["/a/b", "/a/c"]);
    // localStorage is empty in test env, so no recents
    expect(result).toEqual([]);
  });
});
