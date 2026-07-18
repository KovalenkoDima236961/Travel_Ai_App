import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { TripRecapPage } from "@/components/recap";

export default function TripRecapRoute({ params }: { params: { id: string } }) {
  return <ProtectedRoute><TripRecapPage tripId={params.id}/></ProtectedRoute>;
}
