import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { formatMoney } from "@/lib/budget/format";
import type { UnassignedCost } from "@/types/cost-splitting";

type UnassignedCostsPanelProps = {
  costs: UnassignedCost[];
  canEdit: boolean;
  onEditItemSplit?: (dayNumber: number, itemIndex: number) => void;
  onEditAccommodationSplit?: () => void;
  onAddTraveler?: () => void;
};

export function UnassignedCostsPanel({
  costs,
  canEdit,
  onEditItemSplit,
  onEditAccommodationSplit,
  onAddTraveler
}: UnassignedCostsPanelProps) {
  return (
    <Card>
      <div className="flex items-start justify-between gap-3">
        <h2 className="text-lg font-semibold text-slate-950">Unassigned costs</h2>
        {canEdit && onAddTraveler ? (
          <Button onClick={onAddTraveler} size="sm" type="button" variant="secondary">
            Add traveler
          </Button>
        ) : null}
      </div>

      {costs.length === 0 ? (
        <p className="mt-4 text-sm text-slate-600">All converted estimated costs are allocated.</p>
      ) : (
        <div className="mt-4 divide-y divide-slate-100">
          {costs.map((cost, index) => (
            <div className="flex items-start justify-between gap-4 py-3" key={`${cost.type}-${cost.dayNumber}-${cost.itemIndex}-${index}`}>
              <div>
                <p className="font-medium text-slate-950">{cost.name}</p>
                <p className="mt-1 text-xs text-slate-500">
                  {cost.dayNumber ? `Day ${cost.dayNumber}` : "Accommodation"} · {formatReason(cost.reason)}
                </p>
              </div>
              <div className="shrink-0 text-right">
                <p className="font-semibold text-slate-900">{formatMoney(cost.amount, cost.currency)}</p>
                {canEdit ? (
                  <SplitAction
                    cost={cost}
                    onEditAccommodationSplit={onEditAccommodationSplit}
                    onEditItemSplit={onEditItemSplit}
                  />
                ) : null}
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

function SplitAction({
  cost,
  onEditItemSplit,
  onEditAccommodationSplit
}: {
  cost: UnassignedCost;
  onEditItemSplit?: (dayNumber: number, itemIndex: number) => void;
  onEditAccommodationSplit?: () => void;
}) {
  if (cost.type === "itinerary_item" && cost.dayNumber != null && cost.itemIndex != null && onEditItemSplit) {
    return (
      <Button
        className="mt-2"
        onClick={() => onEditItemSplit(cost.dayNumber ?? 0, cost.itemIndex ?? 0)}
        size="sm"
        type="button"
        variant="ghost"
      >
        Set split rule
      </Button>
    );
  }
  if (cost.type === "accommodation" && onEditAccommodationSplit) {
    return (
      <Button className="mt-2" onClick={onEditAccommodationSplit} size="sm" type="button" variant="ghost">
        Set split rule
      </Button>
    );
  }
  return null;
}

function formatReason(reason: string) {
  return reason
    .split(/[_-]/g)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}
