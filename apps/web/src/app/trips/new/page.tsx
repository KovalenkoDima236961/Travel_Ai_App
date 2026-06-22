import { PageContainer } from "@/components/layout/PageContainer";
import { TripForm } from "@/components/trips/TripForm";

export default function NewTripPage() {
  return (
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
      </div>
      <TripForm />
    </PageContainer>
  );
}
