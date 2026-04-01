export type SessionState =
  | "WORKING"
  | "NEEDS_INPUT"
  | "IDLE"
  | "DEAD"
  | "DONE"
  | "STARTING"
  | "UNKNOWN";

export function isAttentionRequired(state: string): boolean {
  return state === "NEEDS_INPUT";
}

export function countAttention(
  sessions: ReadonlyArray<{ executor_state: string }>,
): number {
  return sessions.filter((s) => isAttentionRequired(s.executor_state)).length;
}

export function formatTitle(attentionCount: number): string {
  return attentionCount > 0
    ? `(${attentionCount}) Crowd Control`
    : "Crowd Control";
}

export function formatSessionCount(count: number): string {
  if (count === 0) return "";
  return `${count} session${count !== 1 ? "s" : ""}`;
}

// Layer 1: Connection dot — is the session alive?
// green = online (has a running tmux window)
// red = crashed (window died unexpectedly)
// gray = offline (ended cleanly)
export function connectionDotColor(state: string): string {
  if (state === "DEAD") return "bg-red-400";
  if (state === "DONE") return "bg-gray-500";
  return "bg-emerald-400";
}

// Layer 2: Activity indicator — what is the session doing? (only for online sessions)
// "spinner" = working, "alert" = needs input, "starting" = starting up, null = idle/no indicator
export type ActivityIndicator = "spinner" | "alert" | "starting" | null;

export function activityIndicator(state: string): ActivityIndicator {
  if (state === "WORKING") return "spinner";
  if (state === "NEEDS_INPUT") return "alert";
  if (state === "STARTING") return "starting";
  return null;
}

const badgeColors: Record<string, string> = {
  WORKING: "bg-yellow-400/15 text-yellow-400",
  NEEDS_INPUT: "bg-red-400/15 text-red-400",
  IDLE: "bg-emerald-400/15 text-emerald-400",
  DEAD: "bg-red-400/15 text-red-400",
  DONE: "bg-gray-600/15 text-gray-500",
  STARTING: "bg-gray-600/15 text-gray-500",
};

export function stateBadgeColor(state: string): string {
  return badgeColors[state] ?? "bg-gray-600/15 text-gray-500";
}

export function isResumable(state: string): boolean {
  return state === "DEAD" || state === "DONE";
}

export function isActiveDetail(detail: string | undefined): boolean {
  if (!detail) return false;
  return !detail.startsWith("done:");
}

const statePriority: Record<string, number> = {
  NEEDS_INPUT: 0,
  WORKING: 1,
  STARTING: 2,
  IDLE: 3,
  DEAD: 4,
  DONE: 5,
};

export function sessionSortKey(s: { executor_state: string; updated_at: number }): number {
  const priority = statePriority[s.executor_state] ?? 3;
  return priority * 1e15 - s.updated_at;
}

export function sortSessions<T extends { executor_state: string; updated_at: number }>(
  sessions: readonly T[],
): T[] {
  return [...sessions].sort((a, b) => sessionSortKey(a) - sessionSortKey(b));
}

export function formatRelativeTime(unixSeconds: number): string {
  const now = Math.floor(Date.now() / 1000);
  const diff = now - unixSeconds;
  if (diff < 5) return "just now";
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

export function isEndedState(state: string): boolean {
  return state === "DEAD" || state === "DONE";
}

export function stripMarkdown(text: string): string {
  return text
    .replace(/\*\*(.+?)\*\*/g, "$1")
    .replace(/\*(.+?)\*/g, "$1")
    .replace(/__(.+?)__/g, "$1")
    .replace(/_(.+?)_/g, "$1")
    .replace(/`(.+?)`/g, "$1")
    .replace(/^#{1,6}\s+/gm, "")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/^[-*+]\s+/gm, "")
    .replace(/^\d+\.\s+/gm, "");
}

export function formatDetail(detail: string, state: string): string {
  if (!detail) return "";
  if (state === "DONE" && (detail === "other" || detail.startsWith("done:"))) {
    return "Session ended";
  }
  if (state === "DEAD" && detail === "window lost") {
    return "Window lost";
  }
  if (detail.startsWith("done:")) {
    const tool = detail.slice(5);
    return formatToolName(tool);
  }
  if (detail === "other" || detail === "waiting") return "";
  return detail;
}

function formatToolName(name: string): string {
  if (name.startsWith("mcp__")) {
    const parts = name.split("__");
    return parts[parts.length - 1]?.replace(/_/g, " ") ?? name;
  }
  return name;
}

export interface FormattedCwd {
  display: string;
  isWorkspace: boolean;
  full: string;
}

function abbreviateHome(path: string): string {
  const homeMatch = path.match(/^\/Users\/[^/]+/);
  return homeMatch ? "~" + path.slice(homeMatch[0].length) : path;
}

export function formatCwd(cwd: string, dir?: string): FormattedCwd {
  if (!cwd) return { display: "", isWorkspace: false, full: "" };

  const wsMarker = ".config/cctl/workspaces/";
  const isWorkspace = cwd.includes(wsMarker);

  if (isWorkspace) {
    const originalDir = dir && !dir.includes(wsMarker) ? dir : undefined;
    const display = originalDir ? abbreviateHome(originalDir) : "workspace";
    return { display, isWorkspace: true, full: originalDir || cwd };
  }

  return { display: abbreviateHome(cwd), isWorkspace: false, full: cwd };
}

export interface SessionGroup<T> {
  project_id: string | null;
  project_name: string | null;
  sessions: readonly T[];
}

/**
 * Group sessions by project. Sessions without a project are grouped under
 * null. Groups are ordered: named projects first (alphabetical), then
 * unassigned.
 */
export function groupByProject<T extends { project_id: string | null }>(
  sessions: readonly T[],
  projects: ReadonlyArray<{ id: string; name: string }>,
): SessionGroup<T>[] {
  const projectMap = new Map(projects.map((p) => [p.id, p.name]));
  const groups = new Map<string | null, T[]>();

  for (const s of sessions) {
    const key = s.project_id;
    const list = groups.get(key);
    if (list) {
      list.push(s);
    } else {
      groups.set(key, [s]);
    }
  }

  const result: SessionGroup<T>[] = [];

  // Include all projects, even empty ones
  for (const p of projects) {
    result.push({
      project_id: p.id,
      project_name: p.name,
      sessions: groups.get(p.id) ?? [],
    });
  }

  // Add groups for project_ids not in the projects list (orphaned references)
  for (const [id, list] of groups) {
    if (id != null && !projectMap.has(id)) {
      result.push({ project_id: id, project_name: id, sessions: list });
    }
  }

  result.sort((a, b) => (a.project_name ?? "").localeCompare(b.project_name ?? ""));

  const unassigned = groups.get(null);
  if (unassigned) {
    result.push({ project_id: null, project_name: null, sessions: unassigned });
  }

  return result;
}
