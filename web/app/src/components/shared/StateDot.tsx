import { connectionDotColor, activityIndicator } from "../../domain/session";

export function StateDot({ state }: { state: string }) {
  const dotColor = connectionDotColor(state);
  const activity = activityIndicator(state);

  return (
    <span className="relative flex items-center justify-center w-4 h-4 shrink-0">
      {/* Connection dot */}
      <span className={`w-2 h-2 rounded-full ${dotColor}`} />

      {/* Activity overlay */}
      {activity === "spinner" && (
        <span className="absolute inset-0 flex items-center justify-center">
          <svg className="w-4 h-4 animate-spin text-yellow-400" viewBox="0 0 16 16" fill="none">
            <circle cx="8" cy="8" r="6.5" stroke="currentColor" strokeWidth="1.5" strokeDasharray="10 30" />
          </svg>
        </span>
      )}
      {activity === "alert" && (
        <span className="absolute -top-0.5 -right-0.5 w-2 h-2 bg-red-500 rounded-full animate-pulse" />
      )}
      {activity === "starting" && (
        <span className="absolute inset-0 flex items-center justify-center">
          <span className="w-3.5 h-3.5 rounded-full border border-gray-400 border-dashed animate-spin" style={{ animationDuration: "2s" }} />
        </span>
      )}
    </span>
  );
}
