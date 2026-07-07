import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { NewTripPageContent } from "@/_pages/new-trip/ui/NewTripPageContent";

export default function NewTripPage() {
  return (
    <ProtectedRoute>
      <NewTripPageContent />
    </ProtectedRoute>
  );
}
