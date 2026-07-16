import type { BudgetConfidenceCoverage } from "@/types/budget-confidence";

type CoverageItem = {
  key: keyof BudgetConfidenceCoverage;
  label: string;
};

const COVERAGE_ITEMS: CoverageItem[] = [
  { key: "transport", label: "Transport" },
  { key: "accommodation", label: "Accommodation" },
  { key: "activities", label: "Activities" },
  { key: "food", label: "Food" },
  { key: "shopping", label: "Shopping" },
  { key: "fuelParkingTolls", label: "Fuel, parking, tolls" },
  { key: "other", label: "Other" }
];

export function BudgetCoverageBreakdown({ coverage }: { coverage: BudgetConfidenceCoverage }) {
  const items = COVERAGE_ITEMS.map((item) => ({ ...item, value: coverage[item.key] })).filter(
    (item) => item.value != null
  );

  if (items.length === 0) {
    return null;
  }

  return (
    <div>
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Coverage</p>
      <div className="mt-2 space-y-2">
        {items.map((item) => (
          <div key={item.key}>
            <div className="flex items-center justify-between gap-3 text-xs">
              <span className="text-slate-600">{item.label}</span>
              <span className="font-medium text-slate-900">{item.value}%</span>
            </div>
            <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-slate-100">
              <div
                className="h-full rounded-full bg-emerald-500"
                style={{ width: `${Math.max(0, Math.min(100, item.value ?? 0))}%` }}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
