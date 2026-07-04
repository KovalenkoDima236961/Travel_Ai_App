"use client";

import Link from "next/link";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { TripForm } from "@/components/trips/TripForm";
import { buttonStyles } from "@/components/ui/Button";

export default function NewTripPage() {
  return (
    <ProtectedRoute>
      <PageContainer className="max-w-4xl">
        <div className="mb-8">
          <p className="text-sm font-semibold uppercase text-primary-700">
            New trip
          </p>
          <h1 className="mt-2 text-3xl font-semibold text-slate-950">
            Create a trip request
          </h1>
          <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-600">
            Add the core trip details. You can generate the itinerary after the trip is
            created.
          </p>
          <Link className={buttonStyles({ variant: "secondary", className: "mt-5" })} href="/templates">
            Start from template
          </Link>
        </div>
        <TripForm />
      </PageContainer>
    </ProtectedRoute>
  );
}
