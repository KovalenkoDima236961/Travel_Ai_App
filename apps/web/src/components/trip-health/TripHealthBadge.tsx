import { levelClasses, levelLabel } from "./health-ui";
import type { TripHealth } from "@/types/trip-health";

type TripHealthBadgeProps = {
  health?: TripHealth | null;
  loading?: boolean;
};

export function TripHealthBadge({ health, loading = false }: TripHealthBadgeProps) {
  if (loading && !health) {
    return (
      <span className="inline-flex items-center rounded-full border border-sand-300 bg-white px-3.5 py-1.5 text-[13px] font-medium text-cocoa-400">
        Health...
      </span>
    );
  }
  if (!health) {
    return null;
  }
  return (
    <a
      href="#health"
      title={health.summary}
      className={`inline-flex items-center gap-2 rounded-full border px-3.5 py-1.5 text-[13px] font-semibold transition hover:brightness-[0.98] ${levelClasses(
        health.level
      )}`}
    >
      <span>{health.score}</span>
      <span className="font-medium">{levelLabel[health.level]}</span>
    </a>
  );
}
