import { statusPillClass } from "@/_pages/ops/ui/opsStyles";

export function AIGenerationStatusBadge({ status }: { status?: string | null }) {
  return <span className={statusPillClass(status ?? "unknown")}>{(status ?? "unknown").replace(/_/g, " ")}</span>;
}
