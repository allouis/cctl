import { useState, useMemo } from "react";
import type { Session, Project } from "../../types";
import { SessionCard } from "./SessionCard";
import { sortSessions, isEndedState, groupByProject } from "../../domain/session";
import { AttentionBanner } from "./AttentionBanner";

interface DashboardViewProps {
  sessions: Session[];
  projects: Project[];
  onSelectSession: (sessionId: string) => void;
}

export function DashboardView({
  sessions,
  projects,
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
  const needsInput = useMemo(
    () => active.filter((s) => s.executor_state === "NEEDS_INPUT"),
    [active],
  );

  const visible = showEnded ? sorted : active;
  const groups = useMemo(
    () => groupByProject(visible, projects),
    [visible, projects],
  );
  const hasProjects = projects.length > 0;

  if (sessions.length === 0) {
    return (
      <div className="h-full overflow-y-auto p-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
        <div className="text-center text-gray-500 py-12 text-sm">
          No active sessions
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto p-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
      {needsInput.length > 0 && (
        <AttentionBanner sessions={needsInput} onSelect={onSelectSession} />
      )}

      {hasProjects ? (
        <div className="space-y-4">
          {groups.map((group) => {
            if (group.sessions.length === 0) return null;
            return (
              <section key={group.project_id ?? "__none"}>
                <h2 className="text-[11px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500 mb-2">
                  {group.project_name ?? "Sessions"}
                </h2>
                <div className="space-y-2">
                  {sortSessions(group.sessions).map((s) => (
                    <SessionCard
                      key={s.session_id}
                      session={s}
                      onSelect={() => onSelectSession(s.session_id)}
                    />
                  ))}
                </div>
              </section>
            );
          })}
        </div>
      ) : (
        <div className="space-y-2">
          {visible.map((s) => (
            <SessionCard
              key={s.session_id}
              session={s}
              onSelect={() => onSelectSession(s.session_id)}
            />
          ))}
        </div>
      )}

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
