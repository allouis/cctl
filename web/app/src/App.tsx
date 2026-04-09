import { useState, useCallback, useEffect } from "react";
import { SessionsProvider, useSessions } from "./context/SessionsContext";
import { useTheme } from "./hooks/useTheme";
import { useNotifications } from "./hooks/useNotifications";
import { Sidebar } from "./components/layout/Sidebar";
import { Header } from "./components/layout/Header";
import type { ViewState, Project } from "./types";
import { DashboardView } from "./components/dashboard/DashboardView";
import { SessionView } from "./components/session/SessionView";
import { NewSessionModal } from "./components/shared/NewSessionModal";
import { RepoSessionModal } from "./components/shared/RepoSessionModal";
import { ErrorBoundary } from "./components/shared/ErrorBoundary";
import { deleteSession, getProjects, getRepos } from "./api/client";

function parseUrl(): ViewState {
  const path = window.location.pathname;
  const match = path.match(/^\/session\/(.+)/);
  if (match) {
    return { kind: "session", sessionId: decodeURIComponent(match[1]!) };
  }
  return { kind: "dashboard" };
}

function viewToPath(view: ViewState): string {
  return view.kind === "session"
    ? `/session/${encodeURIComponent(view.sessionId)}`
    : "/";
}

function Layout() {
  const { sessions, connected, updateKey } = useSessions();
  const { theme, setTheme } = useTheme();
  const { attentionCount } = useNotifications(sessions);
  const [view, setView] = useState<ViewState>(parseUrl);
  const [modalOpen, setModalOpen] = useState(false);
  const [repoModalOpen, setRepoModalOpen] = useState(false);
  const [projects, setProjects] = useState<Project[]>([]);
  const [repos, setRepos] = useState<string[]>([]);

  const refreshProjects = useCallback(() => {
    getProjects().then(setProjects).catch(() => {});
  }, []);

  useEffect(() => {
    refreshProjects();
    getRepos().then(setRepos).catch(() => {});
  }, [refreshProjects]);

  const navigate = useCallback((next: ViewState) => {
    setView(next);
    const path = viewToPath(next);
    if (window.location.pathname !== path) {
      window.history.pushState(null, "", path);
    }
  }, []);

  const showDashboard = useCallback(() => navigate({ kind: "dashboard" }), [navigate]);
  const showSession = useCallback(
    (sessionId: string) => navigate({ kind: "session", sessionId }),
    [navigate],
  );

  const handleKillSession = useCallback(async () => {
    if (view.kind !== "session") return;
    try {
      await deleteSession(view.sessionId);
      navigate({ kind: "dashboard" });
    } catch (e) {
      console.error("kill failed:", e);
    }
  }, [view, navigate]);

  // Browser back/forward
  useEffect(() => {
    function handlePopState() {
      setView(parseUrl());
    }
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  // Escape key: close modal, or go back to dashboard
  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") {
        if (modalOpen || repoModalOpen) {
          setModalOpen(false);
          setRepoModalOpen(false);
        } else if (view.kind === "session") {
          showDashboard();
        }
      }
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [modalOpen, repoModalOpen, view, showDashboard]);

  return (
    <div className="flex h-dvh overflow-hidden">
      {/* Sidebar — desktop only */}
      <aside className="hidden lg:flex w-64 bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-700 flex-col shrink-0">
        <Sidebar
          sessions={sessions}
          projects={projects}
          view={view}
          connected={connected}
          attentionCount={attentionCount}
          theme={theme}
          onSetTheme={setTheme}
          onShowDashboard={showDashboard}
          onShowSession={showSession}
          onNewSession={() => repos.length > 0 ? setRepoModalOpen(true) : setModalOpen(true)}
          onProjectsChanged={refreshProjects}
        />
      </aside>

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0">
        <Header
          view={view}
          sessions={sessions}
          onBack={showDashboard}
          onKillSession={handleKillSession}
          onNewSession={() => repos.length > 0 ? setRepoModalOpen(true) : setModalOpen(true)}
        />

        <main className="flex-1 overflow-hidden">
          {view.kind === "dashboard" ? (
            <DashboardView
              sessions={sessions}
              onSelectSession={showSession}
            />
          ) : (
            <SessionView
              session={sessions.find((s) => s.session_id === view.sessionId)}
              sessionId={view.sessionId}
              updateKey={updateKey}
            />
          )}
        </main>
      </div>

      <NewSessionModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onCreated={(sessionId) => {
          showSession(sessionId);
          refreshProjects();
        }}
        projects={projects}
        onProjectCreated={refreshProjects}
        repos={repos}
      />

      <RepoSessionModal
        open={repoModalOpen}
        onClose={() => setRepoModalOpen(false)}
        onCreated={(sessionId) => {
          showSession(sessionId);
          refreshProjects();
        }}
        projects={projects}
        onProjectCreated={refreshProjects}
        repos={repos}
      />
    </div>
  );
}

export function App() {
  return (
    <ErrorBoundary>
      <SessionsProvider>
        <Layout />
      </SessionsProvider>
    </ErrorBoundary>
  );
}
