import { describe, it, expect } from "vitest";
import {
  isAttentionRequired,
  countAttention,
  formatTitle,
  formatSessionCount,
  connectionDotColor,
  activityIndicator,
  stateBadgeColor,
  isActiveDetail,
  sortSessions,
  isEndedState,
  stripMarkdown,
  formatDetail,
  formatCwd,
  formatRelativeTime,
  formatRepoName,
  groupByProject,
} from "./session";

describe("isAttentionRequired", () => {
  it("returns true for NEEDS_INPUT", () => {
    expect(isAttentionRequired("NEEDS_INPUT")).toBe(true);
  });

  it("returns false for other states", () => {
    expect(isAttentionRequired("WORKING")).toBe(false);
    expect(isAttentionRequired("IDLE")).toBe(false);
    expect(isAttentionRequired("DONE")).toBe(false);
    expect(isAttentionRequired("STARTING")).toBe(false);
  });
});

describe("countAttention", () => {
  it("counts NEEDS_INPUT sessions", () => {
    const sessions = [
      { executor_state: "NEEDS_INPUT" },
      { executor_state: "WORKING" },
      { executor_state: "NEEDS_INPUT" },
      { executor_state: "IDLE" },
    ];
    expect(countAttention(sessions)).toBe(2);
  });

  it("returns 0 for empty list", () => {
    expect(countAttention([])).toBe(0);
  });

  it("returns 0 when no NEEDS_INPUT sessions", () => {
    expect(countAttention([{ executor_state: "WORKING" }, { executor_state: "DONE" }])).toBe(0);
  });
});

describe("formatTitle", () => {
  it("includes count when attention needed", () => {
    expect(formatTitle(3)).toBe("(3) Crowd Control");
  });

  it("plain title when no attention", () => {
    expect(formatTitle(0)).toBe("Crowd Control");
  });
});

describe("formatSessionCount", () => {
  it("returns empty for 0", () => {
    expect(formatSessionCount(0)).toBe("");
  });

  it("singular for 1", () => {
    expect(formatSessionCount(1)).toBe("1 session");
  });

  it("plural for many", () => {
    expect(formatSessionCount(5)).toBe("5 sessions");
  });
});

describe("connectionDotColor", () => {
  it("returns green for online states", () => {
    expect(connectionDotColor("WORKING")).toBe("bg-emerald-400");
    expect(connectionDotColor("NEEDS_INPUT")).toBe("bg-emerald-400");
    expect(connectionDotColor("IDLE")).toBe("bg-emerald-400");
    expect(connectionDotColor("STARTING")).toBe("bg-emerald-400");
  });

  it("returns red for crashed", () => {
    expect(connectionDotColor("DEAD")).toBe("bg-red-400");
  });

  it("returns gray for finished", () => {
    expect(connectionDotColor("DONE")).toBe("bg-gray-500");
  });
});

describe("activityIndicator", () => {
  it("returns spinner for WORKING", () => {
    expect(activityIndicator("WORKING")).toBe("spinner");
  });

  it("returns alert for NEEDS_INPUT", () => {
    expect(activityIndicator("NEEDS_INPUT")).toBe("alert");
  });

  it("returns starting for STARTING", () => {
    expect(activityIndicator("STARTING")).toBe("starting");
  });

  it("returns null for idle/ended states", () => {
    expect(activityIndicator("IDLE")).toBeNull();
    expect(activityIndicator("DEAD")).toBeNull();
    expect(activityIndicator("DONE")).toBeNull();
  });
});

describe("stateBadgeColor", () => {
  it("returns correct classes for known states", () => {
    expect(stateBadgeColor("WORKING")).toContain("yellow");
    expect(stateBadgeColor("NEEDS_INPUT")).toContain("red");
    expect(stateBadgeColor("IDLE")).toContain("emerald");
    expect(stateBadgeColor("DONE")).toContain("gray");
  });

  it("falls back for unknown state", () => {
    expect(stateBadgeColor("BOGUS")).toContain("gray");
  });
});

describe("sortSessions", () => {
  it("sorts by state priority then recency", () => {
    const sessions = [
      { executor_state: "DONE", updated_at: 100 },
      { executor_state: "NEEDS_INPUT", updated_at: 200 },
      { executor_state: "WORKING", updated_at: 300 },
      { executor_state: "IDLE", updated_at: 150 },
      { executor_state: "DEAD", updated_at: 50 },
    ];
    const sorted = sortSessions(sessions);
    expect(sorted.map((s) => s.executor_state)).toEqual([
      "NEEDS_INPUT",
      "WORKING",
      "IDLE",
      "DEAD",
      "DONE",
    ]);
  });

  it("sorts by recency within same state", () => {
    const sessions = [
      { executor_state: "WORKING", updated_at: 100 },
      { executor_state: "WORKING", updated_at: 300 },
      { executor_state: "WORKING", updated_at: 200 },
    ];
    const sorted = sortSessions(sessions);
    expect(sorted.map((s) => s.updated_at)).toEqual([300, 200, 100]);
  });
});

describe("isEndedState", () => {
  it("returns true for DEAD and DONE", () => {
    expect(isEndedState("DEAD")).toBe(true);
    expect(isEndedState("DONE")).toBe(true);
  });

  it("returns false for active states", () => {
    expect(isEndedState("WORKING")).toBe(false);
    expect(isEndedState("IDLE")).toBe(false);
    expect(isEndedState("NEEDS_INPUT")).toBe(false);
  });
});

describe("stripMarkdown", () => {
  it("strips bold and italic", () => {
    expect(stripMarkdown("**bold** and *italic*")).toBe("bold and italic");
  });

  it("strips inline code", () => {
    expect(stripMarkdown("use `foo()` here")).toBe("use foo() here");
  });

  it("strips links", () => {
    expect(stripMarkdown("[click](http://example.com)")).toBe("click");
  });

  it("strips headings", () => {
    expect(stripMarkdown("## Title")).toBe("Title");
  });

  it("strips list markers", () => {
    expect(stripMarkdown("- item one\n- item two")).toBe("item one\nitem two");
  });
});

describe("formatDetail", () => {
  it("shows Session ended for DONE with other", () => {
    expect(formatDetail("other", "DONE")).toBe("Session ended");
  });

  it("cleans done: prefix", () => {
    expect(formatDetail("done:mcp__playwright__browser_navigate", "WORKING"))
      .toBe("browser navigate");
  });

  it("returns empty for waiting", () => {
    expect(formatDetail("waiting", "IDLE")).toBe("");
  });

  it("passes through normal details", () => {
    expect(formatDetail("$ ls -la", "WORKING")).toBe("$ ls -la");
  });
});

describe("formatCwd", () => {
  it("replaces home dir with ~", () => {
    expect(formatCwd("/Users/egg/Code/project").display).toBe("~/Code/project");
  });

  it("shows original dir with workspace flag", () => {
    const result = formatCwd(
      "/Users/egg/.config/cctl/workspaces/abc-123",
      "/Users/egg/Code/project",
    );
    expect(result.display).toBe("~/Code/project");
    expect(result.isWorkspace).toBe(true);
  });

  it("falls back to 'workspace' when dir is not provided", () => {
    const result = formatCwd("/Users/egg/.config/cctl/workspaces/abc-123");
    expect(result.display).toBe("workspace");
    expect(result.isWorkspace).toBe(true);
  });

  it("handles empty string", () => {
    expect(formatCwd("").display).toBe("");
  });
});

describe("formatRelativeTime", () => {
  const now = () => Math.floor(Date.now() / 1000);

  it("shows just now for recent timestamps", () => {
    expect(formatRelativeTime(now())).toBe("just now");
  });

  it("shows seconds", () => {
    expect(formatRelativeTime(now() - 30)).toBe("30s ago");
  });

  it("shows minutes", () => {
    expect(formatRelativeTime(now() - 120)).toBe("2m ago");
  });

  it("shows hours", () => {
    expect(formatRelativeTime(now() - 7200)).toBe("2h ago");
  });

  it("shows days", () => {
    expect(formatRelativeTime(now() - 172800)).toBe("2d ago");
  });
});

describe("isActiveDetail", () => {
  it("returns true for PreToolUse details", () => {
    expect(isActiveDetail("$ ls -la")).toBe(true);
    expect(isActiveDetail("reading: file.txt")).toBe(true);
    expect(isActiveDetail("Glob: **/*.ts")).toBe(true);
    expect(isActiveDetail("subagent: explore codebase")).toBe(true);
  });

  it("returns false for PostToolUse completion details", () => {
    expect(isActiveDetail("done:Glob")).toBe(false);
    expect(isActiveDetail("done:Read ✓")).toBe(false);
    expect(isActiveDetail("done:Bash ✗")).toBe(false);
  });

  it("returns false for undefined or empty", () => {
    expect(isActiveDetail(undefined)).toBe(false);
    expect(isActiveDetail("")).toBe(false);
  });
});

describe("groupByProject", () => {
  it("groups sessions by project_id", () => {
    const sessions = [
      { project_id: "p1", name: "a" },
      { project_id: "p2", name: "b" },
      { project_id: "p1", name: "c" },
    ];
    const projects = [
      { id: "p1", name: "Alpha" },
      { id: "p2", name: "Beta" },
    ];
    const groups = groupByProject(sessions, projects);
    expect(groups).toHaveLength(2);
    expect(groups[0]!.project_name).toBe("Alpha");
    expect(groups[0]!.sessions).toHaveLength(2);
    expect(groups[1]!.project_name).toBe("Beta");
    expect(groups[1]!.sessions).toHaveLength(1);
  });

  it("puts unassigned sessions last", () => {
    const sessions = [
      { project_id: null, name: "loose" },
      { project_id: "p1", name: "assigned" },
    ];
    const projects = [{ id: "p1", name: "Alpha" }];
    const groups = groupByProject(sessions, projects);
    expect(groups).toHaveLength(2);
    expect(groups[0]!.project_id).toBe("p1");
    expect(groups[1]!.project_id).toBeNull();
  });

  it("handles all unassigned", () => {
    const sessions = [
      { project_id: null, name: "a" },
      { project_id: null, name: "b" },
    ];
    const groups = groupByProject(sessions, []);
    expect(groups).toHaveLength(1);
    expect(groups[0]!.project_id).toBeNull();
    expect(groups[0]!.sessions).toHaveLength(2);
  });

  it("includes empty projects", () => {
    const sessions = [{ project_id: "p1", name: "a" }];
    const projects = [
      { id: "p1", name: "Alpha" },
      { id: "p2", name: "Beta" },
    ];
    const groups = groupByProject(sessions, projects);
    expect(groups).toHaveLength(2);
    expect(groups[0]!.project_name).toBe("Alpha");
    expect(groups[0]!.sessions).toHaveLength(1);
    expect(groups[1]!.project_name).toBe("Beta");
    expect(groups[1]!.sessions).toHaveLength(0);
  });

  it("handles empty sessions", () => {
    const groups = groupByProject([], []);
    expect(groups).toHaveLength(0);
  });
});

describe("formatRepoName", () => {
  it("returns basename of dir", () => {
    expect(formatRepoName("/home/agent/cctl")).toBe("cctl");
  });

  it("uses dir over cwd when dir is not a workspace path", () => {
    expect(
      formatRepoName("/home/agent/ghost", "/home/agent/.config/cctl/workspaces/abc"),
    ).toBe("ghost");
  });

  it("falls back to cwd basename when dir is a workspace path", () => {
    expect(
      formatRepoName(
        "/home/agent/.config/cctl/workspaces/abc",
        "/home/agent/ghost",
      ),
    ).toBe("ghost");
  });

  it("returns empty string for empty input", () => {
    expect(formatRepoName("")).toBe("");
  });
});

describe("abbreviateHome (via formatCwd)", () => {
  it("replaces /home/user with ~", () => {
    expect(formatCwd("/home/agent/code/project").display).toBe("~/code/project");
  });

  it("replaces /Users/user with ~", () => {
    expect(formatCwd("/Users/egg/Code/project").display).toBe("~/Code/project");
  });
});
