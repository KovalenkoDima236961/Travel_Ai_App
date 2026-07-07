import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { OfflineTripsPageContent } from "@/_pages/offline-trips/ui/OfflineTripsPageContent";

export default function OfflineTripsPage() {
  return (
    <ProtectedRoute>
      <OfflineTripsPageContent />
    </ProtectedRoute>
  );
}
