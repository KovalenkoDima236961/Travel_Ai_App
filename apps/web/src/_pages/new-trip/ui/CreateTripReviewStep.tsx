import type { TripRoute } from "@/entities/route/model";

type CreateTripReviewStepProps = {
  destination: string;
  days: number;
  startDate: string;
  travelers: number;
  budgetAmount?: number;
  budgetCurrency: string;
  interests: string[];
  pace: string;
  outputLanguage: string;
  route: TripRoute | null;
  workspaceWarnings: string[];
};

export function CreateTripReviewStep({
  destination,
  days,
  startDate,
  travelers,
  budgetAmount,
  budgetCurrency,
  interests,
  pace,
  outputLanguage,
  route,
  workspaceWarnings
}: CreateTripReviewStepProps) {
  const routeStops = route?.stops
    .map((stop) => stop.destination.trim())
    .filter(Boolean)
    .join(" → ");
  const summary = [
    ["Destination", routeStops || destination || "Not set"],
    ["Dates", startDate ? `${formatDate(startDate)} · ${days} day${days === 1 ? "" : "s"}` : `${days} day${days === 1 ? "" : "s"}`],
    ["Travelers", `${travelers} ${travelers === 1 ? "traveler" : "travelers"}`],
    ["Budget", budgetAmount != null ? `${budgetAmount.toLocaleString()} ${budgetCurrency}` : "No budget set"],
    ["Style", `${pace} pace${interests.length ? ` · ${interests.join(", ")}` : ""}`],
    ["Plan language", outputLanguage.toUpperCase()]
  ];

  return (
    <section aria-labelledby="trip-review-heading">
      <h2 id="trip-review-heading" className="font-newsreader text-[22px] font-semibold text-cocoa-900">
        Review your trip
      </h2>
      <p className="mt-2 text-[14px] leading-[1.6] text-cocoa-500">
        Check the essentials before creating your trip. You can still edit these details later.
      </p>

      <dl className="mt-5 grid gap-3 sm:grid-cols-2">
        {summary.map(([label, value]) => (
          <div key={label} className="rounded-xl border border-sand-300 bg-sand-50 px-4 py-3">
            <dt className="text-[12px] font-semibold uppercase tracking-[0.06em] text-[#A08D78]">
              {label}
            </dt>
            <dd className="mt-1 text-[14px] font-medium text-cocoa-800">{value}</dd>
          </div>
        ))}
      </dl>

      {workspaceWarnings.length > 0 ? (
        <div className="mt-5 rounded-xl border border-[#EAD9B8] bg-[#FDF7E8] p-4 text-[13.5px] text-[#7A5727]">
          <p className="font-semibold">Workspace policy notes</p>
          <ul className="mt-2 list-disc space-y-1 pl-5">
            {workspaceWarnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="mt-5 rounded-xl border border-[#DCE8DD] bg-[#F2F7F1] p-4 text-[13.5px] leading-[1.6] text-[#38543F]">
        <p className="font-semibold">What happens next</p>
        <p className="mt-1">
          We&apos;ll save this trip, then queue an AI itinerary that uses your dates, preferences,
          route, and budget. You can leave the page while it works.
        </p>
      </div>
    </section>
  );
}

function formatDate(value: string) {
  const parsed = new Date(`${value}T12:00:00`);
  return Number.isNaN(parsed.getTime())
    ? value
    : new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", year: "numeric" }).format(parsed);
}
