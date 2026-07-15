import { HealthCategoryIcon } from "./HealthCategoryIcon";
import { HealthSeverityBadge } from "./HealthSeverityBadge";
import { categoryLabel, scoreBarClass } from "./health-ui";
import type { TripHealthCategorySummary } from "@/types/trip-health";

export function TripHealthCategoryGrid({
  categories
}: {
  categories: TripHealthCategorySummary[];
}) {
  if (categories.length === 0) {
    return null;
  }
  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex items-center justify-between gap-3">
        <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">
          Category Scores
        </h2>
      </div>
      <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {categories.map((category) => (
          <div
            key={category.category}
            className="rounded-[14px] border border-sand-200 bg-sand-50 p-4"
          >
            <div className="flex items-start justify-between gap-3">
              <div className="flex min-w-0 items-center gap-3">
                <HealthCategoryIcon category={category.category} />
                <div className="min-w-0">
                  <p className="truncate text-[14px] font-semibold text-cocoa-900">
                    {categoryLabel[category.category]}
                  </p>
                  <p className="text-[12px] text-cocoa-400">
                    {category.openIssueCount} open{" "}
                    {category.openIssueCount === 1 ? "issue" : "issues"}
                  </p>
                </div>
              </div>
              <HealthSeverityBadge severity={category.highestSeverity} />
            </div>
            <div className="mt-4 flex items-center gap-3">
              <div className="h-2 flex-1 overflow-hidden rounded-full bg-sand-200">
                <div
                  className={`h-full rounded-full ${scoreBarClass(category.score)}`}
                  style={{ width: `${Math.max(0, Math.min(category.score, 100))}%` }}
                />
              </div>
              <span className="w-9 text-right text-[13px] font-semibold text-cocoa-700">
                {category.score}
              </span>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
