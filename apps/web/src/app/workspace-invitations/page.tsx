import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { WorkspaceInvitationsPageContent } from "@/_pages/workspace-invitations/ui/WorkspaceInvitationsPageContent";

export default function WorkspaceInvitationsPage() {
  return (
    <ProtectedRoute>
      <WorkspaceInvitationsPageContent />
    </ProtectedRoute>
  );
}
