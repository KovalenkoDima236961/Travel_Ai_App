import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { TripCostAnalyticsPageContent } from "@/_pages/trip-cost-analytics/ui/TripCostAnalyticsPageContent";

export default function TripCostAnalyticsPage() {
  return (
    <ProtectedRoute>
      <TripCostAnalyticsPageContent />
    </ProtectedRoute>
  );
}
