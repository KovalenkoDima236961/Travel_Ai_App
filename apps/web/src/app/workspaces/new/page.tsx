import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { NewWorkspacePageContent } from "@/_pages/new-workspace/ui/NewWorkspacePageContent";

export default function NewWorkspacePage() {
  return (
    <ProtectedRoute>
      <NewWorkspacePageContent />
    </ProtectedRoute>
  );
}
