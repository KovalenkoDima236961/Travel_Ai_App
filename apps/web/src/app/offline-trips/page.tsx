"use client";

import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { useAuth } from "@/components/auth/AuthProvider";
import { PageContainer } from "@/components/layout/PageContainer";
import { OfflineTripsList } from "@/components/offline/OfflineTripsList";

export default function OfflineTripsPage() {
  return (
    <ProtectedRoute>
      <OfflineTripsPageContent />
    </ProtectedRoute>
  );
}

function OfflineTripsPageContent() {
  const { user } = useAuth();

  return (
    <PageContainer>
      <div className="mb-8">
        <p className="text-sm font-semibold uppercase text-primary-700">Offline</p>
        <h1 className="mt-2 text-3xl font-semibold text-slate-950">Offline trips</h1>
        <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-600">
          View cached trips, review unsynced itinerary drafts, and manage offline storage on this device.
        </p>
      </div>

      {user?.id ? <OfflineTripsList userId={user.id} /> : null}
    </PageContainer>
  );
}
