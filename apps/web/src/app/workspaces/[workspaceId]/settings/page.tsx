import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { WorkspaceSettingsPageContent } from "@/_pages/workspace-settings/ui/WorkspaceSettingsPageContent";

export default function WorkspaceSettingsPage() {
  return (
    <ProtectedRoute>
      <WorkspaceSettingsPageContent />
    </ProtectedRoute>
  );
}
