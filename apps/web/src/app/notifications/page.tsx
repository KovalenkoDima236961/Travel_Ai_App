import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { NotificationsPageContent } from "@/_pages/notifications/ui/NotificationsPageContent";

export default function NotificationsPage() {
  return (
    <ProtectedRoute>
      <NotificationsPageContent />
    </ProtectedRoute>
  );
}
