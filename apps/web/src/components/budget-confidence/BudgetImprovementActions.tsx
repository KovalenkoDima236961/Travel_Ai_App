import type { BudgetConfidenceRecommendation } from "@/types/budget-confidence";

export function BudgetImprovementActions({
  recommendations
}: {
  recommendations: BudgetConfidenceRecommendation[];
}) {
  if (recommendations.length === 0) {
    return null;
  }

  return (
    <div>
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
        Recommended actions
      </p>
      <div className="mt-2 flex flex-wrap gap-2">
        {recommendations.slice(0, 4).map((recommendation) => (
          <a
            className="inline-flex min-h-9 items-center rounded-md border border-slate-300 bg-white px-3 text-xs font-medium text-slate-800 hover:bg-slate-50"
            href={localTripHref(recommendation.href)}
            key={recommendation.id}
          >
            {recommendation.label}
          </a>
        ))}
      </div>
    </div>
  );
}

function localTripHref(href: string) {
  const tab = new URLSearchParams(href.split("?")[1] ?? "").get("tab");
  return tab ? `#${tab === "route" ? "route" : tab}` : href;
}
