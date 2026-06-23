import { Card } from "@/components/ui/Card";
import { getOpeningStatus } from "@/lib/itinerary/opening-hours-utils";
import type { Itinerary } from "@/types/trip";

type OpeningHoursWarningsProps = {
  itinerary: Itinerary;
  startDate?: string | null;
};

type OpeningWarning = {
  key: string;
  dayNumber: number;
  time: string;
  placeName: string;
  label: string;
};

export function OpeningHoursWarnings({ itinerary, startDate }: OpeningHoursWarningsProps) {
  const warnings = collectOpeningHoursWarnings(itinerary, startDate);

  if (warnings.length === 0) {
    return null;
  }

  return (
    <Card className="border-amber-200 bg-amber-50 shadow-none">
      <h2 className="text-lg font-semibold text-amber-950">Opening hours warnings</h2>
      <ul className="mt-3 space-y-2 text-sm text-amber-900">
        {warnings.map((warning) => (
          <li key={warning.key}>
            <span className="font-semibold">
              Day {warning.dayNumber} · {warning.time} · {warning.placeName}
            </span>{" "}
            - {warning.label}
          </li>
        ))}
      </ul>
    </Card>
  );
}

function collectOpeningHoursWarnings(
  itinerary: Itinerary,
  startDate?: string | null
): OpeningWarning[] {
  return (itinerary.days ?? []).flatMap((day, dayIndex) => {
    const dayNumber = day.day || dayIndex + 1;

    return (day.items ?? []).flatMap((item, itemIndex) => {
      const openingHours = item.place?.openingHours;
      if (!openingHours || openingHours.length === 0) {
        return [];
      }

      const status = getOpeningStatus({
        startDate,
        dayNumber,
        itemTime: item.time,
        openingHours
      });
      if (status.status !== "closed") {
        return [];
      }

      return [
        {
          key: `${dayNumber}-${itemIndex}-${item.place?.providerPlaceId ?? item.name}`,
          dayNumber,
          time: item.time,
          placeName: item.place?.name || item.name,
          label: status.label
        }
      ];
    });
  });
}
