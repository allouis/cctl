import { stateBadgeColor } from "../../domain/session";

export function StateBadge({ state }: { state: string }) {
  const colors = stateBadgeColor(state);
  return (
    <span
      className={`inline-block px-2 py-0.5 rounded-full text-[11px] font-bold uppercase tracking-wide ${colors}`}
    >
      {state}
    </span>
  );
}
