import { levelClasses, levelLabel } from "./readiness-ui";
import type { ReadinessLevel } from "@/types/group-readiness";

export function ReadinessScoreBadge({ score, level }: { score: number; level: ReadinessLevel }) {
  return (
    <span
      className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 text-[12px] font-semibold ${levelClasses(
        level
      )}`}
    >
      <span>{score}</span>
      <span>{levelLabel[level]}</span>
    </span>
  );
}

