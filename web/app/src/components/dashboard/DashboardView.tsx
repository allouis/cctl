import { useState, useMemo } from "react";
import type { Session } from "../../types";
import { SessionCard } from "./SessionCard";
import { sortSessions, isEndedState } from "../../domain/session";

interface DashboardViewProps {
  sessions: Session[];
  onSelectSession: (sessionId: string) => void;
}

export function DashboardView({
  sessions,
  onSelectSession,
}: DashboardViewProps) {
  const [showEnded, setShowEnded] = useState(false);

  const sorted = useMemo(() => sortSessions(sessions), [sessions]);
  const active = useMemo(
    () => sorted.filter((s) => !isEndedState(s.executor_state)),
    [sorted],
  );
  const ended = useMemo(
    () => sorted.filter((s) => isEndedState(s.executor_state)),
    [sorted],
  );

  if (sessions.length === 0) {
    return (
      <div className="h-full overflow-y-auto p-4">
        <div className="text-center text-gray-500 py-12 text-sm">
          No active sessions
        </div>
      </div>
    );
  }

  const visible = showEnded ? sorted : active;

  return (
    <div className="h-full overflow-y-auto p-4">
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
        {visible.map((s) => (
          <SessionCard
            key={s.session_id}
            session={s}
            onSelect={() => onSelectSession(s.session_id)}
          />
        ))}
      </div>
      {ended.length > 0 && (
        <button
          onClick={() => setShowEnded(!showEnded)}
          className="mt-3 text-xs text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
        >
          {showEnded
            ? "Hide ended sessions"
            : `Show ${ended.length} ended session${ended.length !== 1 ? "s" : ""}`}
        </button>
      )}
    </div>
  );
}
