import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { WorkspaceBudgetsPageContent } from "@/_pages/workspace-budgets/ui/WorkspaceBudgetsPageContent";

export default function WorkspaceBudgetsPage() {
  return (
    <ProtectedRoute>
      <WorkspaceBudgetsPageContent />
    </ProtectedRoute>
  );
}
