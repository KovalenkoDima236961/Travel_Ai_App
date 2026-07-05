import { CostSplitExportMenu } from "@/components/cost-splitting/CostSplitExportMenu";
import { CostSplitSummaryCards } from "@/components/cost-splitting/CostSplitSummaryCards";
import { PerTravelerCostTable } from "@/components/cost-splitting/PerTravelerCostTable";
import { TravelersPanel } from "@/components/cost-splitting/TravelersPanel";
import { UnassignedCostsPanel } from "@/components/cost-splitting/UnassignedCostsPanel";
import { Card } from "@/components/ui/Card";
import type {
  CostSplittingSummary,
  TripTraveler
} from "@/types/cost-splitting";
import type { Trip } from "@/types/trip";

type CostSplittingPanelProps = {
  trip: Trip;
  travelers: TripTraveler[];
  travelersLoading?: boolean;
  summary?: CostSplittingSummary | null;
  summaryLoading?: boolean;
  canEdit: boolean;
  offline?: boolean;
  onEditItemSplit?: (dayNumber: number, itemIndex: number) => void;
  onEditAccommodationSplit?: () => void;
};

export function CostSplittingPanel({
  trip,
  travelers,
  travelersLoading = false,
  summary,
  summaryLoading = false,
  canEdit,
  offline = false,
  onEditItemSplit,
  onEditAccommodationSplit
}: CostSplittingPanelProps) {
  const currency = summary?.currency ?? trip.budgetCurrency ?? "EUR";

  return (
    <section className="space-y-4" id="cost-splitting">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-xl font-semibold text-slate-950">Cost Split</h2>
          <p className="mt-1 text-sm leading-6 text-slate-600">
            Estimated planning allocations by traveler, category, and day.
          </p>
        </div>
        {summary ? (
          <CostSplitExportMenu
            summary={summary}
            title={`${trip.destination || "Trip"} cost split`}
          />
        ) : null}
      </div>

      {offline ? (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
          Editing split rules requires internet. Cached split data appears here after it has loaded online.
        </div>
      ) : null}

      {summaryLoading ? (
        <div className="rounded-lg border border-slate-200 bg-white p-4 text-sm text-slate-600">
          Loading cost split summary...
        </div>
      ) : null}

      {summary ? <CostSplitSummaryCards summary={summary} /> : null}

      <div className="grid gap-4 xl:grid-cols-[22rem_minmax(0,1fr)]">
        <TravelersPanel
          canEdit={canEdit && !offline}
          currency={currency}
          isLoading={travelersLoading}
          summary={summary}
          travelers={travelers}
          tripId={trip.id}
        />
        {summary ? (
          <PerTravelerCostTable summary={summary} />
        ) : (
          <Card>
            <p className="text-sm text-slate-600">Cost split summary is unavailable.</p>
          </Card>
        )}
      </div>

      {summary ? (
        <UnassignedCostsPanel
          canEdit={canEdit && !offline}
          costs={summary.unassignedCosts}
          onEditAccommodationSplit={onEditAccommodationSplit}
          onEditItemSplit={onEditItemSplit}
        />
      ) : null}

      {summary?.warnings.length ? (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
          <p className="font-medium">Warnings</p>
          <ul className="mt-2 list-disc space-y-1 pl-5">
            {summary.warnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </div>
      ) : null}

      <p className="text-xs leading-5 text-slate-500">
        Estimated planning costs only. This is not a payment request, invoice,
        accounting record, or settlement calculation.
      </p>
    </section>
  );
}
