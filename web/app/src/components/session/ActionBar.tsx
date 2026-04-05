import type { Session } from "../../types";
import { Spinner } from "../shared/Spinner";
import { useActionState } from "../../hooks/useActionState";
import { sendText, takeoverSession } from "../../api/client";

export function ActionBar({ session }: { session: Session }) {
  const approve = useActionState();
  const deny = useActionState();
  const takeover = useActionState();

  if (session.executor_state !== "NEEDS_INPUT") return null;

  if (session.attached) {
    return (
      <div className="border-t border-gray-200 dark:border-gray-700 border-l-3 border-l-amber-400 bg-amber-500/8 px-4 py-2.5 pb-[max(0.625rem,env(safe-area-inset-bottom))] flex items-center gap-3">
        <span className="flex-1 text-xs font-medium text-gray-600 dark:text-gray-300 truncate">
          {session.preview || "Permission required"} — Attached via tmux
        </span>
        <button
          onClick={() =>
            takeover.execute(() => takeoverSession(session.session_id), { stayLoading: true })
          }
          disabled={takeover.loading}
          className="px-3 py-1 text-xs font-semibold text-amber-400 bg-amber-500/15 hover:bg-amber-500/25 rounded-md transition-colors"
        >
          {takeover.loading ? <Spinner /> : "Take control"}
        </button>
        {takeover.error && (
          <span className="text-xs text-red-500">{takeover.error}</span>
        )}
      </div>
    );
  }

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 border-l-3 border-l-red-400 bg-red-500/8 px-4 py-2.5 pb-[max(0.625rem,env(safe-area-inset-bottom))] flex items-center gap-3">
      <span className="flex-1 text-xs font-medium text-gray-600 dark:text-gray-300 truncate">
        {session.preview || "Permission required"}
      </span>
      <button
        onClick={() => approve.execute(() => sendText(session.session_id, "y"))}
        disabled={approve.loading || deny.loading}
        className="px-3 py-1 text-xs font-semibold text-emerald-400 bg-emerald-500/15 hover:bg-emerald-500/25 rounded-md transition-colors"
      >
        {approve.loading ? <Spinner /> : "Approve"}
      </button>
      <button
        onClick={() => deny.execute(() => sendText(session.session_id, "n"))}
        disabled={approve.loading || deny.loading}
        className="px-3 py-1 text-xs font-semibold text-red-400 bg-red-500/15 hover:bg-red-500/25 rounded-md transition-colors"
      >
        {deny.loading ? <Spinner /> : "Deny"}
      </button>
    </div>
  );
}
