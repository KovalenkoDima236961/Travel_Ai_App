import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { AIGenerationTraceDetailPage } from "@/components/ops/ai-generations/AIGenerationTraceDetailPage";

export default async function AIGenerationTracePage({
  params
}: {
  params: Promise<{ traceId: string }>;
}) {
  const { traceId } = await params;
  return <ProtectedRoute><AIGenerationTraceDetailPage traceId={traceId} /></ProtectedRoute>;
}
