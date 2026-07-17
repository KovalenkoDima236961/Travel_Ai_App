import { AIGenerationContextSummary } from "./AIGenerationContextSummary";
export function ValidationSummaryPanel({ value }: { value?: Record<string, unknown> | null }) { return <AIGenerationContextSummary title="Validation summary" value={value} />; }
