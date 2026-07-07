import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { WorkspaceBudgetDetailPageContent } from "@/_pages/workspace-budget-detail/ui/WorkspaceBudgetDetailPageContent";

export default function WorkspaceBudgetDetailPage() {
  return (
    <ProtectedRoute>
      <WorkspaceBudgetDetailPageContent />
    </ProtectedRoute>
  );
}
