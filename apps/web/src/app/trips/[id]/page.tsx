"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { PageContainer } from "@/components/layout/PageContainer";
import { GenerateItineraryButton } from "@/components/trips/GenerateItineraryButton";
import { ItineraryView } from "@/components/trips/ItineraryView";
import { TripStatusBadge } from "@/components/trips/TripStatusBadge";
import { Card } from "@/components/ui/Card";
import { buttonStyles } from "@/components/ui/Button";
import { getTrip, tripKeys } from "@/lib/api/trips";
import {
  formatBudget,
  formatDate,
  formatInterestLabel,
  formatPaceLabel
} from "@/lib/utils";

export default function TripDetailPage() {
  const params = useParams<{ id: string }>();
  const tripId = params.id;

  const tripQuery = useQuery({
    queryKey: tripKeys.detail(tripId),
    queryFn: () => getTrip(tripId),
    enabled: Boolean(tripId),
    refetchInterval: (query) =>
      query.state.data?.status === "PROCESSING" ? 3000 : false
  });

  if (tripQuery.isPending) {
    return (
      <PageContainer>
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading trip...
        </div>
      </PageContainer>
    );
  }

  if (tripQuery.isError) {
    return (
      <PageContainer>
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {tripQuery.error instanceof Error ? tripQuery.error.message : "Could not load trip."}
        </div>
        <Link className={buttonStyles({ variant: "secondary", className: "mt-5" })} href="/trips">
          Back to trips
        </Link>
      </PageContainer>
    );
  }

  const trip = tripQuery.data;
  const canGenerate = trip.status === "DRAFT" || trip.status === "FAILED";

  return (
    <PageContainer>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <Link className="text-sm font-medium text-primary-700 hover:text-primary-600" href="/trips">
            Back to trips
          </Link>
          <div className="mt-3 flex flex-wrap items-center gap-3">
            <h1 className="text-3xl font-semibold text-slate-950">{trip.destination}</h1>
            <TripStatusBadge status={trip.status} />
          </div>
        </div>
        {canGenerate ? <GenerateItineraryButton tripId={trip.id} /> : null}
      </div>

      <div className="grid gap-6 lg:grid-cols-[22rem_minmax(0,1fr)]">
        <Card>
          <h2 className="text-lg font-semibold text-slate-950">Trip details</h2>
          <dl className="mt-5 space-y-4 text-sm">
            <DetailRow label="Start date" value={trip.startDate ? formatDate(trip.startDate) : "Not set"} />
            <DetailRow label="Duration" value={`${trip.days} ${trip.days === 1 ? "day" : "days"}`} />
            <DetailRow label="Travelers" value={`${trip.travelers}`} />
            <DetailRow label="Budget" value={formatBudget(trip.budgetAmount, trip.budgetCurrency)} />
            <DetailRow label="Pace" value={formatPaceLabel(trip.pace)} />
            <DetailRow
              label="Created"
              value={formatDate(trip.createdAt, {
                dateStyle: "medium",
                timeStyle: "short"
              })}
            />
          </dl>
          <div className="mt-6">
            <p className="text-sm font-medium text-slate-700">Interests</p>
            <div className="mt-2 flex flex-wrap gap-2">
              {trip.interests.length > 0 ? (
                trip.interests.map((interest) => (
                  <span
                    key={interest}
                    className="rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-medium text-slate-700"
                  >
                    {formatInterestLabel(interest)}
                  </span>
                ))
              ) : (
                <span className="text-sm text-slate-500">No interests selected</span>
              )}
            </div>
          </div>
        </Card>

        <section className="min-w-0">
          {trip.status === "PROCESSING" ? (
            <div className="rounded-lg border border-amber-200 bg-amber-50 p-6 text-sm text-amber-900">
              The itinerary is being generated. This page will refresh while processing.
            </div>
          ) : null}

          {trip.status === "COMPLETED" && trip.itinerary ? (
            <ItineraryView itinerary={trip.itinerary} currency={trip.budgetCurrency} />
          ) : null}

          {trip.status === "COMPLETED" && !trip.itinerary ? (
            <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
              This trip is completed, but no itinerary was returned.
            </div>
          ) : null}

          {(trip.status === "DRAFT" || trip.status === "FAILED") && !trip.itinerary ? (
            <div className="rounded-lg border border-slate-200 bg-white p-6">
              <h2 className="text-lg font-semibold text-slate-950">No itinerary yet</h2>
              <p className="mt-2 text-sm leading-6 text-slate-600">
                Generate an itinerary when the Trip Service and AI Planning Service are
                running.
              </p>
            </div>
          ) : null}

          {(trip.status === "DRAFT" || trip.status === "FAILED") && trip.itinerary ? (
            <ItineraryView itinerary={trip.itinerary} currency={trip.budgetCurrency} />
          ) : null}
        </section>
      </div>
    </PageContainer>
  );
}

type DetailRowProps = {
  label: string;
  value: string;
};

function DetailRow({ label, value }: DetailRowProps) {
  return (
    <div className="flex items-start justify-between gap-4">
      <dt className="text-slate-500">{label}</dt>
      <dd className="text-right font-medium text-slate-800">{value}</dd>
    </div>
  );
}
