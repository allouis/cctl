import { useMemo, useState } from "react";
import type { Session, Project, Theme, ViewState } from "../../types";
import { StateDot } from "../shared/StateDot";
import { sortSessions, formatRelativeTime, groupByProject } from "../../domain/session";
import { createProject } from "../../api/client";

const ThemeIcons = {
  light: (
    <svg
      className="w-4 h-4"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth="1.5"
      stroke="currentColor"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M12 3v2.25m6.364.386-1.591 1.591M21 12h-2.25m-.386 6.364-1.591-1.591M12 18.75V21m-4.773-4.227-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0Z"
      />
    </svg>
  ),
  system: (
    <svg
      className="w-4 h-4"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth="1.5"
      stroke="currentColor"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M9 17.25v1.007a3 3 0 0 1-.879 2.122L7.5 21h9l-.621-.621A3 3 0 0 1 15 18.257V17.25m6-12V15a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 15V5.25A2.25 2.25 0 0 1 5.25 3h13.5A2.25 2.25 0 0 1 21 5.25Z"
      />
    </svg>
  ),
  dark: (
    <svg
      className="w-4 h-4"
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth="1.5"
      stroke="currentColor"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M21.752 15.002A9.72 9.72 0 0 1 18 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 0 0 3 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 0 0 9.002-5.998Z"
      />
    </svg>
  ),
};

function SessionItem({ session: s, active, onClick }: { session: Session; active: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={`w-full flex items-center gap-2.5 px-3 py-1.5 text-sm rounded-md hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors ${
        active ? "bg-gray-100 dark:bg-gray-800" : ""
      }`}
    >
      <StateDot state={s.executor_state} />
      <span className="truncate text-gray-700 dark:text-gray-200">
        {s.name}
      </span>
      <span className="ml-auto text-[10px] text-gray-400 dark:text-gray-500 shrink-0">
        {formatRelativeTime(s.updated_at)}
      </span>
    </button>
  );
}

interface SidebarProps {
  sessions: Session[];
  projects: Project[];
  view: ViewState;
  connected: boolean;
  attentionCount: number;
  theme: Theme;
  onSetTheme: (t: Theme) => void;
  onShowDashboard: () => void;
  onShowSession: (sessionId: string) => void;
  onNewSession: () => void;
  onProjectsChanged: () => void;
  onOpenSystemPrompt: () => void;
}

export function Sidebar({
  sessions: unsortedSessions,
  projects,
  view,
  connected,
  attentionCount,
  theme,
  onSetTheme,
  onShowDashboard,
  onShowSession,
  onNewSession,
  onProjectsChanged,
  onOpenSystemPrompt,
}: SidebarProps) {
  const sessions = useMemo(() => sortSessions(unsortedSessions), [unsortedSessions]);
  const groups = useMemo(() => groupByProject(sessions, projects), [sessions, projects]);
  const hasProjects = projects.length > 0;
  const [creatingProject, setCreatingProject] = useState(false);
  const [newProjectName, setNewProjectName] = useState("");
  return (
    <>
      <div className="flex items-center gap-3 px-4 py-3 pt-[max(0.75rem,env(safe-area-inset-top))] border-b border-gray-200 dark:border-gray-700">
        <div
          className={`w-2.5 h-2.5 rounded-full shrink-0 ${connected ? "bg-emerald-500" : "bg-red-500"}`}
          title={connected ? "Connected" : "Disconnected"}
        />
        <h1 className="text-sm font-semibold text-gray-900 dark:text-gray-100 tracking-tight">
          Crowd Control
        </h1>
        {attentionCount > 0 && (
          <span className="ml-auto min-w-5 h-5 inline-flex items-center justify-center px-1 text-[10px] font-bold text-white bg-red-500 rounded-full">
            {attentionCount}
          </span>
        )}
      </div>

      <nav className="px-2 py-2">
        <button
          onClick={onShowDashboard}
          className={`w-full flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-gray-900 dark:text-gray-100 rounded-md ${
            view.kind === "dashboard" ? "bg-gray-100 dark:bg-gray-800" : ""
          }`}
        >
          <svg
            className="w-4 h-4 text-gray-500 dark:text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth="1.5"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M3.75 6A2.25 2.25 0 0 1 6 3.75h2.25A2.25 2.25 0 0 1 10.5 6v2.25a2.25 2.25 0 0 1-2.25 2.25H6a2.25 2.25 0 0 1-2.25-2.25V6ZM3.75 15.75A2.25 2.25 0 0 1 6 13.5h2.25a2.25 2.25 0 0 1 2.25 2.25V18a2.25 2.25 0 0 1-2.25 2.25H6A2.25 2.25 0 0 1 3.75 18v-2.25ZM13.5 6a2.25 2.25 0 0 1 2.25-2.25H18A2.25 2.25 0 0 1 20.25 6v2.25A2.25 2.25 0 0 1 18 10.5h-2.25a2.25 2.25 0 0 1-2.25-2.25V6ZM13.5 15.75a2.25 2.25 0 0 1 2.25-2.25H18a2.25 2.25 0 0 1 2.25 2.25V18A2.25 2.25 0 0 1 18 20.25h-2.25a2.25 2.25 0 0 1-2.25-2.25v-2.25Z"
            />
          </svg>
          Dashboard
        </button>
      </nav>

      <div className="flex-1 overflow-y-auto px-2 py-1">
        {creatingProject ? (
          <div className="px-3 py-1.5">
            <input
              autoFocus
              type="text"
              value={newProjectName}
              onChange={(e) => setNewProjectName(e.target.value)}
              onKeyDown={async (e) => {
                if (e.key === "Enter" && newProjectName.trim()) {
                  await createProject(newProjectName.trim());
                  setNewProjectName("");
                  setCreatingProject(false);
                  onProjectsChanged();
                } else if (e.key === "Escape") {
                  setNewProjectName("");
                  setCreatingProject(false);
                }
              }}
              onBlur={() => {
                setNewProjectName("");
                setCreatingProject(false);
              }}
              placeholder="Project name"
              className="w-full bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-xs border border-gray-300 dark:border-gray-700 rounded px-2 py-1 outline-none focus:border-indigo-500"
            />
          </div>
        ) : (
          <button
            onClick={() => setCreatingProject(true)}
            className="w-full flex items-center gap-2 px-3 py-1 text-xs text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          >
            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
            </svg>
            New Project
          </button>
        )}
        {hasProjects ? (
          groups.map((group) => (
            <div key={group.project_id ?? "_unassigned"} className="mb-1">
              <div className="px-3 py-1.5 text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider">
                {group.project_name ?? "Sessions"}
              </div>
              <div className="space-y-0.5">
                {group.sessions.map((s) => (
                  <SessionItem
                    key={s.session_id}
                    session={s}
                    active={view.kind === "session" && view.sessionId === s.session_id}
                    onClick={() => onShowSession(s.session_id)}
                  />
                ))}
              </div>
            </div>
          ))
        ) : (
          <>
            <div className="px-3 py-1.5 text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider">
              Sessions
            </div>
            <div className="space-y-0.5">
              {sessions.map((s) => (
                <SessionItem
                  key={s.session_id}
                  session={s}
                  active={view.kind === "session" && view.sessionId === s.session_id}
                  onClick={() => onShowSession(s.session_id)}
                />
              ))}
            </div>
          </>
        )}
      </div>

      <div className="border-t border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-center gap-1 px-3 py-2">
          {(["light", "system", "dark"] as const).map((t) => (
            <button
              key={t}
              onClick={() => onSetTheme(t)}
              className={`p-1.5 rounded-md transition-colors ${
                theme === t
                  ? "text-indigo-500 dark:text-indigo-400"
                  : "text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300"
              }`}
              title={t.charAt(0).toUpperCase() + t.slice(1)}
            >
              {ThemeIcons[t]}
            </button>
          ))}
          <div className="w-px h-4 bg-gray-200 dark:bg-gray-700 mx-1" />
          <button
            onClick={onOpenSystemPrompt}
            className="p-1.5 rounded-md text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
            title="System Prompt"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth="1.5" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.325.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 0 1 1.37.49l1.296 2.247a1.125 1.125 0 0 1-.26 1.431l-1.003.827c-.293.241-.438.613-.43.992a7.723 7.723 0 0 1 0 .255c-.008.378.137.75.43.991l1.004.827c.424.35.534.955.26 1.43l-1.298 2.247a1.125 1.125 0 0 1-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.47 6.47 0 0 1-.22.128c-.331.183-.581.495-.644.869l-.213 1.281c-.09.543-.56.94-1.11.94h-2.594c-.55 0-1.019-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 0 1-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 0 1-1.369-.49l-1.297-2.247a1.125 1.125 0 0 1 .26-1.431l1.004-.827c.292-.24.437-.613.43-.991a6.932 6.932 0 0 1 0-.255c.007-.38-.138-.751-.43-.992l-1.004-.827a1.125 1.125 0 0 1-.26-1.43l1.297-2.247a1.125 1.125 0 0 1 1.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.086.22-.128.332-.183.582-.495.644-.869l.214-1.28Z" />
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
            </svg>
          </button>
        </div>
        <div className="px-3 pb-3">
          <button
            onClick={onNewSession}
            className="w-full flex items-center justify-center gap-2 px-3 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-md transition-colors"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth="2"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M12 4.5v15m7.5-7.5h-15"
              />
            </svg>
            New Session
          </button>
        </div>
      </div>
    </>
  );
}
