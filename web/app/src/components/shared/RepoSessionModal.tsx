import { useState, useRef, useEffect } from "react";
import { Modal } from "./Modal";
import { createSession, createProject } from "../../api/client";
import { searchRepos, getRecentRepos, addRecentRepo } from "../../domain/repos";
import type { Project } from "../../types";

interface RepoSessionModalProps {
  open: boolean;
  onClose: () => void;
  onCreated: (sessionId: string) => void;
  projects: Project[];
  onProjectCreated: () => void;
  repos: string[];
}

type Step = "pick-repo" | "configure";

export function RepoSessionModal({
  open,
  onClose,
  onCreated,
  projects,
  onProjectCreated,
  repos,
}: RepoSessionModalProps) {
  const [step, setStep] = useState<Step>("pick-repo");
  const [query, setQuery] = useState("");
  const [selectedRepo, setSelectedRepo] = useState("");
  const [customDir, setCustomDir] = useState("");
  const [name, setName] = useState("");
  const [autoApprove, setAutoApprove] = useState(true);
  const [projectId, setProjectId] = useState("");
  const [newProjectName, setNewProjectName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [highlightIdx, setHighlightIdx] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  const nameRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLUListElement>(null);

  const recents = getRecentRepos(repos);
  const showRecents = !query && recents.length > 0;
  const results = query ? searchRepos(repos, query, 8) : [];

  useEffect(() => {
    if (open) {
      setStep("pick-repo");
      setQuery("");
      setSelectedRepo("");
      setCustomDir("");
      setName("");
      setAutoApprove(true);
      setProjectId("");
      setNewProjectName("");
      setError(null);
      setCreating(false);
      setHighlightIdx(0);
      setTimeout(() => searchRef.current?.focus(), 0);
    }
  }, [open]);

  useEffect(() => {
    if (step === "configure") {
      setTimeout(() => nameRef.current?.focus(), 0);
    }
  }, [step]);

  useEffect(() => {
    setHighlightIdx(0);
  }, [query]);

  // Scroll highlighted item into view
  useEffect(() => {
    const list = listRef.current;
    if (!list) return;
    const item = list.children[highlightIdx] as HTMLElement | undefined;
    item?.scrollIntoView({ block: "nearest" });
  }, [highlightIdx]);

  function selectRepo(path: string) {
    setSelectedRepo(path);
    const basename = path.split("/").pop() || "";
    setName(basename);
    setStep("configure");
  }

  function handleSearchKeyDown(e: React.KeyboardEvent) {
    const items = showRecents ? recents : results.map((r) => r.path);
    if (items.length === 0) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      setHighlightIdx((i) => Math.min(i + 1, items.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setHighlightIdx((i) => Math.max(i - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      const item = items[highlightIdx];
      if (item) selectRepo(item);
    }
  }

  async function handleCreate() {
    if (!name.trim()) {
      setError("Name is required");
      return;
    }

    setCreating(true);
    setError(null);

    try {
      let resolvedProjectId = projectId || undefined;

      if (projectId === "__new" && newProjectName.trim()) {
        const p = await createProject(newProjectName.trim());
        resolvedProjectId = p.id;
        onProjectCreated();
      } else if (projectId === "__new") {
        resolvedProjectId = undefined;
      }

      const dir = selectedRepo || customDir.trim();
      const result = await createSession({
        name: name.trim(),
        dir,
        safe: !autoApprove,
        project_id: resolvedProjectId,
      });
      if (selectedRepo) addRecentRepo(selectedRepo);
      onClose();
      onCreated(result.session_id);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
      setCreating(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose}>
      <div className="relative bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-xl shadow-xl w-full max-w-md">
        {step === "pick-repo" ? (
          <>
            <div className="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                Choose a repo
              </h3>
            </div>
            <div className="px-5 py-4 space-y-3">
              <div className="relative">
                <svg
                  className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth="2"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z"
                  />
                </svg>
                <input
                  ref={searchRef}
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  onKeyDown={handleSearchKeyDown}
                  placeholder="Search repos..."
                  autoComplete="off"
                  className="w-full bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm border border-gray-300 dark:border-gray-700 rounded-md pl-9 pr-3 py-2 outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 placeholder-gray-400 dark:placeholder-gray-600"
                />
              </div>

              <ul
                ref={listRef}
                className="max-h-64 overflow-y-auto -mx-5"
              >
                {showRecents && (
                  <>
                    <li className="px-5 py-1.5">
                      <span className="text-[11px] font-medium uppercase tracking-wider text-gray-400 dark:text-gray-500">
                        Recent
                      </span>
                    </li>
                    {recents.map((path, i) => (
                      <RepoItem
                        key={path}
                        path={path}
                        highlighted={i === highlightIdx}
                        onSelect={() => selectRepo(path)}
                        onHover={() => setHighlightIdx(i)}
                      />
                    ))}
                  </>
                )}

                {query && results.length > 0 &&
                  results.map((r, i) => (
                    <RepoItem
                      key={r.path}
                      path={r.path}
                      highlighted={i === highlightIdx}
                      onSelect={() => selectRepo(r.path)}
                      onHover={() => setHighlightIdx(i)}
                    />
                  ))}

                {query && results.length === 0 && (
                  <li className="px-5 py-6 text-center text-sm text-gray-400 dark:text-gray-500">
                    No repos match "{query}"
                  </li>
                )}

                {!query && recents.length === 0 && (
                  <li className="px-5 py-6 text-center text-sm text-gray-400 dark:text-gray-500">
                    Start typing to search {repos.length} repos
                  </li>
                )}
              </ul>
            </div>
            <div className="px-5 py-3 border-t border-gray-200 dark:border-gray-700 flex justify-between items-center">
              <button
                onClick={() => {
                  setSelectedRepo("");
                  setName("");
                  setStep("configure");
                }}
                className="text-xs text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
              >
                Skip — use custom path
              </button>
              <button
                onClick={onClose}
                className="px-3 py-1.5 text-sm font-medium text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 transition-colors"
              >
                Cancel
              </button>
            </div>
          </>
        ) : (
          <>
            <div className="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setStep("pick-repo")}
                  className="p-0.5 -ml-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth="2" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
                  </svg>
                </button>
                <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                  New Session
                </h3>
              </div>
              {selectedRepo && (
                <p className="mt-1 text-xs text-gray-400 dark:text-gray-500 truncate">
                  {selectedRepo}
                </p>
              )}
            </div>
            <div className="px-5 py-4 space-y-3">
              <div>
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                  Name
                </label>
                <input
                  ref={nameRef}
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleCreate();
                  }}
                  placeholder="my-session"
                  autoComplete="off"
                  className="w-full bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm border border-gray-300 dark:border-gray-700 rounded-md px-3 py-1.5 outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 placeholder-gray-400 dark:placeholder-gray-600"
                />
              </div>
              {!selectedRepo && (
                <div>
                  <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                    Working Directory
                  </label>
                  <input
                    type="text"
                    value={customDir}
                    onChange={(e) => setCustomDir(e.target.value)}
                    placeholder="~/code/project"
                    autoComplete="off"
                    className="w-full bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm border border-gray-300 dark:border-gray-700 rounded-md px-3 py-1.5 outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 placeholder-gray-400 dark:placeholder-gray-600"
                  />
                </div>
              )}
              <div>
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                  Project
                </label>
                <select
                  value={projectId}
                  onChange={(e) => setProjectId(e.target.value)}
                  className="w-full bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm border border-gray-300 dark:border-gray-700 rounded-md px-3 py-1.5 outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
                >
                  <option value="">None</option>
                  {projects.map((p) => (
                    <option key={p.id} value={p.id}>{p.name}</option>
                  ))}
                  <option value="__new">+ New project</option>
                </select>
                {projectId === "__new" && (
                  <input
                    type="text"
                    value={newProjectName}
                    onChange={(e) => setNewProjectName(e.target.value)}
                    placeholder="Project name"
                    autoComplete="off"
                    className="mt-1.5 w-full bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm border border-gray-300 dark:border-gray-700 rounded-md px-3 py-1.5 outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 placeholder-gray-400 dark:placeholder-gray-600"
                  />
                )}
              </div>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={autoApprove}
                  onChange={(e) => setAutoApprove(e.target.checked)}
                  className="rounded border-gray-300 dark:border-gray-600 text-indigo-500 focus:ring-indigo-500"
                />
                <span className="text-xs text-gray-600 dark:text-gray-400">
                  Auto-approve tool use
                </span>
              </label>
              {error && <div className="text-xs text-red-500">{error}</div>}
            </div>
            <div className="px-5 py-3 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-2">
              <button
                onClick={onClose}
                className="px-3 py-1.5 text-sm font-medium text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={creating}
                className="px-3 py-1.5 text-sm font-semibold text-white bg-indigo-600 hover:bg-indigo-500 rounded-md transition-colors disabled:opacity-50"
              >
                {creating ? "Creating..." : "Create"}
              </button>
            </div>
          </>
        )}
      </div>
    </Modal>
  );
}

function RepoItem({
  path,
  highlighted,
  onSelect,
  onHover,
}: {
  path: string;
  highlighted: boolean;
  onSelect: () => void;
  onHover: () => void;
}) {
  const basename = path.split("/").pop()!;

  return (
    <li
      onMouseDown={onSelect}
      onMouseEnter={onHover}
      className={`px-5 py-2 cursor-pointer flex items-center justify-between gap-2 ${
        highlighted
          ? "bg-indigo-50 dark:bg-indigo-900/20"
          : "hover:bg-gray-50 dark:hover:bg-gray-800"
      }`}
    >
      <span className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
        {basename}
      </span>
      <span className="text-xs text-gray-400 dark:text-gray-500 truncate shrink-0 max-w-[50%] text-right">
        {path}
      </span>
    </li>
  );
}
