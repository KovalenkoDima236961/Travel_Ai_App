import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { DemoTripPageContent } from "@/_pages/demo-trip/ui/DemoTripPageContent";

export default function DemoTripPage() {
  return <ProtectedRoute><DemoTripPageContent /></ProtectedRoute>;
}
