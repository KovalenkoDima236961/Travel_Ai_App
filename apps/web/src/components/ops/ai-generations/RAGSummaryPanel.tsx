import { AIGenerationContextSummary } from "./AIGenerationContextSummary";
export function RAGSummaryPanel({ value }: { value?: Record<string, unknown> | null }) { return <AIGenerationContextSummary title="RAG summary" value={value} />; }
