import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { TravelDayPage } from "@/components/travel-day";

export default function TravelDayRoute({ params }: { params: { id: string } }) {
  return <ProtectedRoute><TravelDayPage tripId={params.id}/></ProtectedRoute>;
}
