import { AIGenerationContextSummary } from "./AIGenerationContextSummary";
export function RepairAttemptsPanel({ value }: { value?: Record<string, unknown> | null }) { return <AIGenerationContextSummary title="Repair attempts" value={value} />; }
