import type { AIGenerationTraceFilters as Filters } from "@/lib/api/ops";
import { FilterInput, FilterSelect } from "@/_pages/ops/ui/OpsPageParts";
import { OUTLINE_BUTTON } from "@/_pages/ops/ui/opsStyles";

export function AIGenerationTraceFilters({ filters, onChange }: { filters: Filters; onChange: (next: Filters) => void }) {
  const update = (key: keyof Filters, value: string) => onChange({ ...filters, [key]: value || undefined });
  return (
    <div className="flex flex-wrap items-end gap-3">
      <FilterSelect label="Status" value={filters.status ?? ""} onChange={(value) => update("status", value)} options={["", "started", "completed", "completed_with_warnings", "failed", "blocked"]} />
      <FilterSelect label="Type" value={filters.generationType ?? ""} onChange={(value) => update("generationType", value)} options={["", "full_generation", "day_regeneration", "item_regeneration", "quality_improvement_day", "quality_improvement_item", "budget_optimization_day", "template_adaptation", "policy_repair"]} />
      <FilterInput label="Provider / model" value={filters.provider ?? ""} onChange={(value) => update("provider", value)} />
      <FilterInput label="Trip ID" value={filters.tripId ?? ""} onChange={(value) => update("tripId", value)} />
      <FilterInput label="Job ID" value={filters.jobId ?? ""} onChange={(value) => update("jobId", value)} />
      <label className="flex h-[38px] items-center gap-2 text-[13px] text-cocoa-600"><input type="checkbox" checked={Boolean(filters.errorOnly)} onChange={(event) => onChange({ ...filters, errorOnly: event.target.checked || undefined })} /> Errors only</label>
      <button type="button" className={OUTLINE_BUTTON} onClick={() => onChange({})}>Clear</button>
    </div>
  );
}
