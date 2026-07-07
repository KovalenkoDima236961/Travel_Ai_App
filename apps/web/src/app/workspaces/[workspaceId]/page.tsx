import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { WorkspaceDetailPageContent } from "@/_pages/workspace-detail/ui/WorkspaceDetailPageContent";

export default function WorkspaceDetailPage() {
  return (
    <ProtectedRoute>
      <WorkspaceDetailPageContent />
    </ProtectedRoute>
  );
}
