import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { SettingsPageContent } from "@/_pages/settings/ui/SettingsPageContent";

export default function SettingsPage() {
  return (
    <ProtectedRoute>
      <SettingsPageContent />
    </ProtectedRoute>
  );
}
