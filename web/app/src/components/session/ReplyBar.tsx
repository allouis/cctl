import { useState, useRef, useCallback } from "react";
import { Spinner } from "../shared/Spinner";
import { sendText, resumeSession, takeoverSession } from "../../api/client";

const MAX_RETRIES = 5;
const INITIAL_DELAY = 1000;

interface ReplyBarProps {
  sessionId: string;
  sessionState?: string;
  attached?: boolean;
  onSent?: (text: string) => void;
}

export function ReplyBar({ sessionId, sessionState, attached, onSent }: ReplyBarProps) {
  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const [retrying, setRetrying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [resuming, setResuming] = useState(false);
  const abortRef = useRef(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const [takingOver, setTakingOver] = useState(false);
  const isDead = sessionState === "DEAD" || sessionState === "DONE";

  async function handleTakeover() {
    setTakingOver(true);
    setError(null);
    try {
      await takeoverSession(sessionId);
      // Stay in loading state — the SSE update will swap this
      // block for the textarea when attached becomes false.
    } catch (e) {
      setTakingOver(false);
      setError(e instanceof Error ? e.message : "Take control failed");
    }
  }

  const resize = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 200) + "px";
  }, []);

  async function trySend(message: string, retries: number): Promise<{ ok: boolean; confirmed?: boolean }> {
    for (let attempt = 0; attempt <= retries; attempt++) {
      if (abortRef.current) return { ok: false };
      try {
        const result = await sendText(sessionId, message);
        return { ok: true, confirmed: result.confirmed };
      } catch {
        if (attempt < retries) {
          setRetrying(true);
          const delay = INITIAL_DELAY * Math.pow(2, attempt);
          await new Promise((r) => setTimeout(r, delay));
        }
      }
    }
    return { ok: false };
  }

  async function handleSend() {
    const trimmed = text.trim();
    if (!trimmed || sending) return;

    setSending(true);
    setRetrying(false);
    setError(null);
    abortRef.current = false;
    const savedText = trimmed;
    setText("");
    onSent?.(trimmed);
    requestAnimationFrame(() => {
      const el = textareaRef.current;
      if (el) el.style.height = "auto";
    });

    const { ok, confirmed } = await trySend(trimmed, MAX_RETRIES);
    setRetrying(false);
    setSending(false);

    if (!ok && !abortRef.current) {
      setError("Send failed after retries — tap Send to try again");
      setText(savedText);
    } else if (ok && confirmed === false) {
      setError("Sent but delivery not confirmed — Claude may still be starting up");
    }
    textareaRef.current?.focus();
  }

  async function handleResume() {
    setResuming(true);
    setError(null);
    try {
      await resumeSession(sessionId);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Resume failed");
    } finally {
      setResuming(false);
    }
  }

  if (isDead) {
    return (
      <div className="border-t border-gray-200 dark:border-gray-700 px-4 py-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] bg-white dark:bg-gray-950">
        <div className="flex items-center justify-between">
          <span className="text-xs text-gray-400 dark:text-gray-500">
            Session ended
          </span>
          <button
            onClick={handleResume}
            disabled={resuming}
            className="px-3 py-1.5 text-xs font-semibold text-blue-400 bg-blue-500/15 hover:bg-blue-500/25 rounded-md transition-colors"
          >
            {resuming ? <Spinner /> : "Resume"}
          </button>
        </div>
        {error && (
          <div className="text-xs text-red-500 mt-1">{error}</div>
        )}
      </div>
    );
  }

  if (attached) {
    return (
      <div className="border-t border-gray-200 dark:border-gray-700 px-4 py-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] bg-white dark:bg-gray-950">
        <div className="flex items-center justify-between">
          <span className="text-xs text-gray-400 dark:text-gray-500">
            Attached via tmux
          </span>
          <button
            onClick={handleTakeover}
            disabled={takingOver}
            className="px-3 py-1.5 text-xs font-semibold text-amber-400 bg-amber-500/15 hover:bg-amber-500/25 rounded-md transition-colors"
          >
            {takingOver ? <Spinner /> : "Take control"}
          </button>
        </div>
        {error && (
          <div className="text-xs text-red-500 mt-1">{error}</div>
        )}
      </div>
    );
  }

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 px-4 py-2.5 pb-[max(0.625rem,env(safe-area-inset-bottom))] bg-white dark:bg-gray-950">
      {error && (
        <div className="text-xs text-red-500 mb-1.5">{error}</div>
      )}
      <div className="flex gap-2 items-end">
        <textarea
          ref={textareaRef}
          value={text}
          onChange={(e) => {
            setText(e.target.value);
            resize();
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              handleSend();
            }
          }}
          placeholder="Reply to session..."
          autoComplete="off"
          disabled={sending}
          rows={1}
          className="flex-1 bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100 text-sm border border-gray-300 dark:border-gray-700 rounded-md px-3 py-1.5 outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 placeholder-gray-400 dark:placeholder-gray-600 resize-none"
        />
        <button
          onClick={handleSend}
          disabled={sending || !text.trim()}
          className="px-3 py-1.5 text-sm font-semibold text-white bg-indigo-600 hover:bg-indigo-500 rounded-md transition-colors"
        >
          {retrying ? "Retrying..." : sending ? <Spinner /> : "Send"}
        </button>
      </div>
    </div>
  );
}
