import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { WorkspaceCostAnalyticsPageContent } from "@/_pages/workspace-cost-analytics/ui/WorkspaceCostAnalyticsPageContent";

export default function WorkspaceCostAnalyticsPage() {
  return (
    <ProtectedRoute>
      <WorkspaceCostAnalyticsPageContent />
    </ProtectedRoute>
  );
}
