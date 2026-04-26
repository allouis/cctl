import type { Session, Project, TranscriptEntry } from "../types";

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || res.statusText);
  }
  return res.json() as Promise<T>;
}

export function getSessions(): Promise<Session[]> {
  return fetchJSON<Session[]>("/api/sessions");
}

export function getSession(name: string): Promise<Session> {
  return fetchJSON<Session>(`/api/sessions/${encodeURIComponent(name)}`);
}

export function createSession(opts: {
  name: string;
  dir: string;
  safe?: boolean;
  harness?: string;
  project_id?: string;
}): Promise<{ status: string; name: string; session_id: string }> {
  return fetchJSON("/api/sessions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ...opts, dir: opts.dir || "." }),
  });
}

export function updateSessionProject(
  sessionId: string,
  projectId: string | null,
): Promise<{ status: string }> {
  return fetchJSON(`/api/sessions/${encodeURIComponent(sessionId)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ project_id: projectId }),
  });
}

export function deleteSession(sessionId: string): Promise<{ status: string }> {
  return fetchJSON(`/api/sessions/${encodeURIComponent(sessionId)}`, {
    method: "DELETE",
  });
}

export function resumeSession(sessionId: string): Promise<{ status: string }> {
  return fetchJSON(`/api/resume/${encodeURIComponent(sessionId)}`, {
    method: "POST",
  });
}

export function getTranscript(
  sessionId: string,
  limit = 100,
): Promise<TranscriptEntry[]> {
  return fetchJSON<TranscriptEntry[]>(
    `/api/transcript/${encodeURIComponent(sessionId)}?limit=${limit}`,
  );
}

export function takeoverSession(
  sessionId: string,
): Promise<{ status: string }> {
  return fetchJSON(`/api/takeover/${encodeURIComponent(sessionId)}`, {
    method: "POST",
  });
}

export function sendText(
  sessionId: string,
  text: string,
): Promise<{ confirmed: boolean }> {
  return fetchJSON(`/api/send/${encodeURIComponent(sessionId)}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ text }),
  });
}

export function getConfig(): Promise<{ default_harness: string }> {
  return fetchJSON<{ default_harness: string }>("/api/config");
}

export function getRepos(): Promise<string[]> {
  return fetchJSON<string[]>("/api/repos");
}

export function getProjects(): Promise<Project[]> {
  return fetchJSON<Project[]>("/api/projects");
}

export function createProject(
  name: string,
): Promise<Project> {
  return fetchJSON("/api/projects", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
}

export function deleteProject(id: string): Promise<{ status: string }> {
  return fetchJSON(`/api/projects/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export function getSystemPrompt(): Promise<{ content: string }> {
  return fetchJSON<{ content: string }>("/api/system-prompt");
}

export function saveSystemPrompt(
  content: string,
): Promise<{ status: string }> {
  return fetchJSON("/api/system-prompt", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  });
}
