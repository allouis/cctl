export interface SessionSnapshot {
  session_id: string;
  name: string;
  executor_state: string;
  preview: string;
}

export interface StateTransition {
  name: string;
  kind: "needs_input" | "finished" | "ended";
  executor_detail: string;
}

/**
 * Detect notable session state transitions.
 */
export function detectTransitions(
  prevStates: ReadonlyMap<string, string>,
  sessions: readonly SessionSnapshot[],
): StateTransition[] {
  const transitions: StateTransition[] = [];
  for (const s of sessions) {
    const prev = prevStates.get(s.session_id);
    if (!prev) continue;

    if (s.executor_state === "NEEDS_INPUT" && prev !== "NEEDS_INPUT") {
      transitions.push({
        name: s.name,
        kind: "needs_input",
        executor_detail: s.preview || "Permission required",
      });
    } else if (s.executor_state === "IDLE" && prev === "WORKING") {
      transitions.push({
        name: s.name,
        kind: "finished",
        executor_detail: s.preview || "Task complete",
      });
    } else if (s.executor_state === "DONE" && prev === "WORKING") {
      transitions.push({
        name: s.name,
        kind: "ended",
        executor_detail: "Session ended",
      });
    }
  }
  return transitions;
}

/**
 * Build a state snapshot map from a list of sessions, keyed by session_id.
 */
export function buildStateMap(
  sessions: readonly SessionSnapshot[],
): Map<string, string> {
  const map = new Map<string, string>();
  for (const s of sessions) {
    map.set(s.session_id, s.executor_state);
  }
  return map;
}
