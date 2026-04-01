import type { Session } from "../../types";
import { StateBadge } from "../shared/StateBadge";
import { Spinner } from "../shared/Spinner";
import { useActionState } from "../../hooks/useActionState";
import { sendText, resumeSession, takeoverSession } from "../../api/client";
import { isResumable, stripMarkdown, formatDetail, formatRelativeTime } from "../../domain/session";

interface SessionCardProps {
  session: Session;
  onSelect: () => void;
}

export function SessionCard({ session, onSelect }: SessionCardProps) {
  const approve = useActionState();
  const deny = useActionState();
  const resume = useActionState();

  return (
    <div
      onClick={onSelect}
      className="bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-xl p-4 cursor-pointer shadow-sm hover:shadow-md hover:border-gray-300 dark:hover:border-gray-600 transition-all"
    >
      <div className="flex items-center gap-2 mb-2">
        <span className="font-semibold text-sm text-gray-900 dark:text-gray-100">
          {session.name}
        </span>
        <StateBadge state={session.executor_state} />
        <span className="ml-auto text-[10px] text-gray-400 dark:text-gray-500">
          {formatRelativeTime(session.updated_at)}
        </span>
      </div>
      {formatDetail(session.executor_detail, session.executor_state) && (
        <div className="text-xs text-gray-500 truncate">
          {formatDetail(session.executor_detail, session.executor_state)}
        </div>
      )}
      {session.preview && (
        <div className="text-xs text-gray-500 mt-2 line-clamp-2">
          {stripMarkdown(session.preview)}
        </div>
      )}
      {session.executor_state === "NEEDS_INPUT" && (
        session.attached ? (
          <div className="flex items-center gap-2 mt-3">
            <span className="text-[10px] text-amber-400">Attached via tmux</span>
            <button
              onClick={(e) => {
                e.stopPropagation();
                approve.execute(() => takeoverSession(session.session_id), { stayLoading: true });
              }}
              disabled={approve.loading}
              className="px-2.5 py-1 text-xs font-semibold text-amber-400 bg-amber-500/15 hover:bg-amber-500/25 rounded transition-colors"
            >
              {approve.loading ? <Spinner /> : "Take control"}
            </button>
          </div>
        ) : (
          <div className="flex gap-2 mt-3">
            <button
              onClick={(e) => {
                e.stopPropagation();
                approve.execute(() => sendText(session.session_id, "y"));
              }}
              disabled={approve.loading || deny.loading}
              className="px-2.5 py-1 text-xs font-semibold text-emerald-400 bg-emerald-500/15 hover:bg-emerald-500/25 rounded transition-colors"
            >
              {approve.loading ? <Spinner /> : "Approve"}
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                deny.execute(() => sendText(session.session_id, "n"));
              }}
              disabled={approve.loading || deny.loading}
              className="px-2.5 py-1 text-xs font-semibold text-red-400 bg-red-500/15 hover:bg-red-500/25 rounded transition-colors"
            >
              {deny.loading ? <Spinner /> : "Deny"}
            </button>
          </div>
        )
      )}
      {isResumable(session.executor_state) && (
        <div className="flex gap-2 mt-3">
          <button
            onClick={(e) => {
              e.stopPropagation();
              resume.execute(() => resumeSession(session.session_id));
            }}
            disabled={resume.loading}
            className="px-2.5 py-1 text-xs font-semibold text-blue-400 bg-blue-500/15 hover:bg-blue-500/25 rounded transition-colors"
          >
            {resume.loading ? <Spinner /> : "Resume"}
          </button>
        </div>
      )}
      {(approve.error || deny.error || resume.error) && (
        <div className="text-xs text-red-500 mt-1">
          {approve.error || deny.error || resume.error}
        </div>
      )}
    </div>
  );
}
