import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { WorkspacesPageContent } from "@/_pages/workspaces/ui/WorkspacesPageContent";

export default function WorkspacesPage() {
  return (
    <ProtectedRoute>
      <WorkspacesPageContent />
    </ProtectedRoute>
  );
}
