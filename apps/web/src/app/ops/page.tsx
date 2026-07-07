import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { OpsPageContent } from "@/_pages/ops/ui/OpsPageContent";

export default function OpsPage() {
  return (
    <ProtectedRoute>
      <OpsPageContent />
    </ProtectedRoute>
  );
}
