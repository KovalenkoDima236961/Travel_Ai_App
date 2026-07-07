import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { WorkspaceTemplatesPageContent } from "@/_pages/workspace-templates/ui/WorkspaceTemplatesPageContent";

export default function WorkspaceTemplatesPage() {
  return (
    <ProtectedRoute>
      <WorkspaceTemplatesPageContent />
    </ProtectedRoute>
  );
}
