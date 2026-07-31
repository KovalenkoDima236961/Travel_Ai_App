import { notFound } from "next/navigation";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { TripDetailPageContent } from "@/_pages/trip-detail/ui/TripDetailPageContent";
import { isTripWorkspaceSection } from "@/lib/trip-workspace/navigation";

export default async function TripWorkspaceSectionPage({
  params
}: {
  params: Promise<{ id: string; section: string }>;
}) {
  const { section } = await params;
  if (!isTripWorkspaceSection(section)) {
    notFound();
  }
  return (
    <ProtectedRoute>
      <TripDetailPageContent />
    </ProtectedRoute>
  );
}
