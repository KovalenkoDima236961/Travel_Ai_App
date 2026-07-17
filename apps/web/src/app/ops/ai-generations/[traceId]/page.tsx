import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { AIGenerationTraceDetailPage } from "@/components/ops/ai-generations/AIGenerationTraceDetailPage";

export default function AIGenerationTracePage({ params }: { params: { traceId: string } }) {
  return <ProtectedRoute><AIGenerationTraceDetailPage traceId={params.traceId} /></ProtectedRoute>;
}
