import type { Session, ViewState } from "../../types";
import { formatSessionCount, formatCwd } from "../../domain/session";

interface HeaderProps {
  view: ViewState;
  sessions: Session[];
  onBack: () => void;
  onKillSession: () => void;
  onNewSession: () => void;
  onOpenSystemPrompt: () => void;
}

export function Header({
  view,
  sessions,
  onBack,
  onKillSession,
  onNewSession,
  onOpenSystemPrompt,
}: HeaderProps) {
  const session =
    view.kind === "session"
      ? sessions.find((s) => s.session_id === view.sessionId)
      : null;

  const cwd = session ? formatCwd(session.cwd, session.dir) : null;

  return (
    <header className="flex items-center gap-3 px-4 py-3 pt-[max(0.75rem,env(safe-area-inset-top))] border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-950 shrink-0">
      {view.kind === "session" && (
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
      )}

      <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100 truncate">
        {view.kind === "dashboard" ? "Dashboard" : session?.name ?? ""}
      </h2>

      {view.kind === "dashboard" && sessions.length > 0 && (
        <span className="text-xs text-gray-400 dark:text-gray-500">
          {formatSessionCount(sessions.length)}
        </span>
      )}

      <div className="ml-auto flex items-center gap-2">
        <button
          onClick={onOpenSystemPrompt}
          className="lg:hidden p-1 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          title="System Prompt"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" strokeWidth="1.5" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z" />
            <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
          </svg>
        </button>

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
