import type { Session, ViewState } from "../../types";
import { formatSessionCount, formatCwd } from "../../domain/session";

interface HeaderProps {
  view: ViewState;
  sessions: Session[];
  onBack: () => void;
  onKillSession: () => void;
  onNewSession: () => void;
}

export function Header({
  view,
  sessions,
  onBack,
  onKillSession,
  onNewSession,
}: HeaderProps) {
  const session =
    view.kind === "session"
      ? sessions.find((s) => s.session_id === view.sessionId)
      : null;

  const cwd = session ? formatCwd(session.cwd, session.dir) : null;

  return (
    <header className="flex items-center gap-3 px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-950 shrink-0">
      {view.kind === "session" ? (
        <button
          onClick={onBack}
          className="lg:hidden p-1 -ml-1 text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100"
        >
          <svg
            className="w-5 h-5"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth="1.5"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M10.5 19.5 3 12m0 0 7.5-7.5M3 12h18"
            />
          </svg>
        </button>
      ) : (
        <span className="lg:hidden text-sm font-semibold text-gray-900 dark:text-gray-100">
          Crowd Control
        </span>
      )}

      <h2 className={`text-sm font-semibold text-gray-900 dark:text-gray-100 truncate ${view.kind === "dashboard" ? "lg:block" : ""}`}>
        {view.kind === "dashboard" ? "Dashboard" : session?.name ?? ""}
      </h2>

      {view.kind === "dashboard" && sessions.length > 0 && (
        <span className="text-xs text-gray-400 dark:text-gray-500 hidden lg:inline">
          {formatSessionCount(sessions.length)}
        </span>
      )}

      <div className="ml-auto flex items-center gap-2">
        {view.kind === "dashboard" && (
          <button
            onClick={onNewSession}
            className="lg:hidden px-2.5 py-1 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 rounded-md transition-colors"
          >
            + New
          </button>
        )}

        {view.kind === "session" && session && cwd && (
          <>
            <span
              className="text-xs text-gray-400 dark:text-gray-500 truncate max-w-xs hidden sm:inline"
              title={cwd.full}
            >
              {cwd.display}
              {cwd.isWorkspace && (
                <span className="ml-1.5 text-indigo-400/80">(workspace)</span>
              )}
            </span>
            <button
              onClick={onKillSession}
              className="px-2 py-0.5 text-xs text-gray-400 dark:text-gray-500 hover:text-red-400 transition-colors"
            >
              Kill session
            </button>
          </>
        )}
      </div>
    </header>
  );
}
