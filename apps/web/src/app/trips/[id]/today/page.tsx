import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { TravelDayPage } from "@/components/travel-day";

export default async function TravelDayRoute({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <ProtectedRoute><TravelDayPage tripId={id}/></ProtectedRoute>;
}
