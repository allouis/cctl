import type { Session } from "../../types";
import { formatRepoName } from "../../domain/session";

interface AttentionBannerProps {
  sessions: Session[];
  onSelect: (sessionId: string) => void;
}

export function AttentionBanner({ sessions, onSelect }: AttentionBannerProps) {
  return (
    <div className="mb-4 rounded-xl border border-red-200 dark:border-red-900/50 bg-red-50 dark:bg-red-950/30 p-3">
      <div className="text-[11px] font-semibold uppercase tracking-wider text-red-400 mb-2">
        Needs attention
      </div>
      <div className="space-y-1.5">
        {sessions.map((s) => (
          <button
            key={s.session_id}
            onClick={() => onSelect(s.session_id)}
            className="w-full flex items-center gap-2 rounded-lg px-2.5 py-1.5 text-left hover:bg-red-100 dark:hover:bg-red-900/20 transition-colors"
          >
            <span className="w-1.5 h-1.5 rounded-full bg-red-400 shrink-0 animate-pulse" />
            <span className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
              {s.name}
            </span>
            {formatRepoName(s.dir, s.cwd) && (
              <span className="text-[11px] text-gray-400 dark:text-gray-500 truncate ml-auto">
                {formatRepoName(s.dir, s.cwd)}
              </span>
            )}
          </button>
        ))}
      </div>
    </div>
  );
}
