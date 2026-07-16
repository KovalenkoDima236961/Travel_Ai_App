import { categoryLabel } from "./readiness-ui";
import type { ReadinessCategory } from "@/types/group-readiness";

export function ReadinessCategoryBadge({ category }: { category: ReadinessCategory }) {
  return (
    <span className="inline-flex rounded-full border border-sand-300 bg-sand-50 px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.08em] text-cocoa-500">
      {categoryLabel(category)}
    </span>
  );
}

