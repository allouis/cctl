export interface Session {
  session_id: string;
  name: string;
  parent_id: string | null;
  project_id: string | null;
  work_state: string;
  executor_state: string;
  executor_detail: string;
  tool: string;
  preview: string;
  dir: string;
  cwd: string;
  last_event: string;
  transcript_path: string;
  conversation_id: string;
  window_id: string;
  workspace: string;
  prompt: string;
  safe: boolean;
  created_at: number;
  updated_at: number;
  attached: boolean;
}

export interface Project {
  id: string;
  name: string;
  created_at: number;
}

export interface TranscriptEntry {
  role: "user" | "assistant" | "tool_use" | "tool_result";
  text: string;
  full_text?: string;
  is_error?: boolean;
  tool_use_id?: string;
}

export type Theme = "light" | "dark" | "system";

export type ViewState =
  | { kind: "dashboard" }
  | { kind: "session"; sessionId: string };
