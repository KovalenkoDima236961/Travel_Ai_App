import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { TripDetailPageContent } from "@/_pages/trip-detail/ui/TripDetailPageContent";

export default function TripDetailPage() {
  return (
    <ProtectedRoute>
      <TripDetailPageContent />
    </ProtectedRoute>
  );
}
