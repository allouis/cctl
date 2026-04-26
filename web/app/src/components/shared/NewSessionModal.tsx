import { useState, useRef, useEffect } from "react";
import { Modal } from "./Modal";
import { createSession, createProject } from "../../api/client";
import type { Project } from "../../types";

interface NewSessionModalProps {
  open: boolean;
  onClose: () => void;
  onCreated: (sessionId: string) => void;
  projects: Project[];
  onProjectCreated: () => void;
  repos: string[];
  defaultHarness: string;
}

export function NewSessionModal({ open, onClose, onCreated, projects, onProjectCreated, repos, defaultHarness }: NewSessionModalProps) {
  const [name, setName] = useState("");
  const [dir, setDir] = useState("");
  const [harness, setHarness] = useState(defaultHarness);
  const [autoApprove, setAutoApprove] = useState(true);
  const [projectId, setProjectId] = useState<string>("");
  const [newProjectName, setNewProjectName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [showRepos, setShowRepos] = useState(false);
  const [highlightIdx, setHighlightIdx] = useState(-1);
  const nameRef = useRef<HTMLInputElement>(null);
  const dirRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLUListElement>(null);

  const filteredRepos = repos.filter((r) =>
    r.toLowerCase().includes(dir.toLowerCase()),
  );

  useEffect(() => {
    if (open) {
      setName("");
      setDir("");
      setHarness(defaultHarness);
      setAutoApprove(true);
      setProjectId("");
      setNewProjectName("");
      setError(null);
      setCreating(false);
      setShowRepos(false);
      setHighlightIdx(-1);
      setTimeout(() => nameRef.current?.focus(), 0);
    }
  }, [open]);

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

      const result = await createSession({
        name: name.trim(),
        dir: dir.trim(),
        safe: harness === "claude" ? !autoApprove : false,
        harness,
        project_id: resolvedProjectId,
      });
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
        <div className="px-5 py-4 border-b border-gray-200 dark:border-gray-700">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
            New Session
          </h3>
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
              placeholder="my-project"
              autoComplete="off"
              className="w-full bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm border border-gray-300 dark:border-gray-700 rounded-md px-3 py-1.5 outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 placeholder-gray-400 dark:placeholder-gray-600"
            />
          </div>
          <div className="relative">
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
              Working Directory
            </label>
            <input
              ref={dirRef}
              type="text"
              value={dir}
              onChange={(e) => {
                setDir(e.target.value);
                setHighlightIdx(-1);
                if (repos.length > 0) setShowRepos(true);
              }}
              onFocus={() => {
                if (repos.length > 0) setShowRepos(true);
              }}
              onBlur={() => {
                // Delay so click on dropdown registers first
                setTimeout(() => setShowRepos(false), 150);
              }}
              onKeyDown={(e) => {
                if (showRepos && filteredRepos.length > 0) {
                  if (e.key === "ArrowDown") {
                    e.preventDefault();
                    setHighlightIdx((i) => Math.min(i + 1, filteredRepos.length - 1));
                  } else if (e.key === "ArrowUp") {
                    e.preventDefault();
                    setHighlightIdx((i) => Math.max(i - 1, 0));
                  } else if (e.key === "Enter" && highlightIdx >= 0) {
                    e.preventDefault();
                    setDir(filteredRepos[highlightIdx]!);
                    setShowRepos(false);
                    setHighlightIdx(-1);
                    return;
                  } else if (e.key === "Escape") {
                    e.stopPropagation();
                    setShowRepos(false);
                    return;
                  }
                }
                if (e.key === "Enter") handleCreate();
              }}
              placeholder={repos.length > 0 ? "Search repos or type a path..." : "~/Code/project"}
              autoComplete="off"
              className="w-full bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm border border-gray-300 dark:border-gray-700 rounded-md px-3 py-1.5 outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 placeholder-gray-400 dark:placeholder-gray-600"
            />
            {showRepos && filteredRepos.length > 0 && (
              <ul
                ref={dropdownRef}
                className="absolute z-10 left-0 right-0 mt-1 max-h-40 overflow-y-auto bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md shadow-lg"
              >
                {filteredRepos.map((repo, i) => (
                  <li
                    key={repo}
                    onMouseDown={() => {
                      setDir(repo);
                      setShowRepos(false);
                      setHighlightIdx(-1);
                    }}
                    className={`px-3 py-1.5 cursor-pointer text-sm ${
                      i === highlightIdx
                        ? "bg-indigo-50 dark:bg-indigo-900/30"
                        : "hover:bg-gray-50 dark:hover:bg-gray-700"
                    }`}
                  >
                    <span className="text-gray-900 dark:text-gray-100">
                      {repo.split("/").pop()}
                    </span>
                    <span className="ml-2 text-xs text-gray-400 dark:text-gray-500">
                      {repo}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
              Harness
            </label>
            <select
              value={harness}
              onChange={(e) => setHarness(e.target.value)}
              className="w-full bg-gray-50 dark:bg-gray-800 text-gray-900 dark:text-gray-100 text-sm border border-gray-300 dark:border-gray-700 rounded-md px-3 py-1.5 outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
            >
              <option value="claude">Claude Code</option>
              <option value="pi">Pi</option>
            </select>
          </div>
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
          {harness === "claude" && (
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
          )}
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
            className="px-3 py-1.5 text-sm font-semibold text-white bg-indigo-600 hover:bg-indigo-500 rounded-md transition-colors"
          >
            {creating ? "Creating..." : "Create"}
          </button>
        </div>
      </div>
    </Modal>
  );
}
