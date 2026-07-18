import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { TripRecapPage } from "@/components/recap";

export default async function TripRecapRoute({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <ProtectedRoute><TripRecapPage tripId={id}/></ProtectedRoute>;
}
