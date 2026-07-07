import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { TripsPageContent } from "@/_pages/trips/ui/TripsPageContent";

export default function TripsPage() {
  return (
    <ProtectedRoute>
      <TripsPageContent />
    </ProtectedRoute>
  );
}
