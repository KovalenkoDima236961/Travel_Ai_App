import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { WorkspaceApprovalsPageContent } from "@/_pages/workspace-approvals/ui/WorkspaceApprovalsPageContent";

export default function WorkspaceApprovalsPage() {
  return (
    <ProtectedRoute>
      <WorkspaceApprovalsPageContent />
    </ProtectedRoute>
  );
}
