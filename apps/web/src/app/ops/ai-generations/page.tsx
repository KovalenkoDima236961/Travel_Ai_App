import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { AIGenerationsPageContent } from "@/components/ops/ai-generations/AIGenerationsPageContent";

export default function AIGenerationsPage() {
  return <ProtectedRoute><AIGenerationsPageContent /></ProtectedRoute>;
}
