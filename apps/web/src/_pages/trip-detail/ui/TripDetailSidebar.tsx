import { formatMoney, getCostAmount, getCostCurrency } from "@/entities/budget/model";
import { formatDate } from "@/lib/utils";
import { BudgetSummaryCard } from "./BudgetSummaryCard";
import { SectionNav } from "./SectionNav";
import { PlusIcon } from "./icons";
import type { BudgetSummary } from "@/entities/budget/model";
import type { TripTraveler } from "@/entities/cost-splitting/model";
import type { Trip } from "@/entities/trip/model";
import type { NavigationGroup } from "@/types/trip-command-center";

const AVATAR_COLORS = [
  "bg-[#3E6B5A] text-[#EFF5F1]",
  "bg-clay text-sand-100",
  "bg-[#8F3D24] text-sand-100",
  "bg-cocoa-900 text-sand-100"
];

type TripDetailSidebarProps = {
  trip: Trip;
  tripId: string;
  budgetSummary: BudgetSummary | null;
  budgetCurrency: string;
  budgetLoading: boolean;
  canMutateTrip: boolean;
  navigationGroups?: NavigationGroup[];
  optimizationDisabled: boolean;
  onOpenBudgetOptimization: (dayNumber: number) => void;
  perPersonAverage?: { amount: number; currency: string } | null;
  travelers: TripTraveler[];
};

/**
 * Warm left rail for the redesigned Trip Detail screen: section nav plus the
 * budget / stay / travelers summary cards from the mock. The deep interactive
 * panels (budget editing, accommodation form, sharing, cost-split) render in the
 * page's anchored "tools" region so all logic is preserved.
 */
export function TripDetailSidebar({
  trip,
  tripId,
  budgetSummary,
  budgetCurrency,
  budgetLoading,
  canMutateTrip,
  navigationGroups,
  optimizationDisabled,
  onOpenBudgetOptimization,
  perPersonAverage,
  travelers
}: TripDetailSidebarProps) {
  const accommodation = trip.accommodation ?? null;
  const stayCost = getCostAmount(accommodation?.estimatedCost);
  const stayCostCurrency = getCostCurrency(accommodation?.estimatedCost) ?? budgetCurrency;
  const stayMeta = [
    accommodation?.address,
    formatStayDates(accommodation?.checkInDate, accommodation?.checkOutDate),
    stayCost != null ? formatMoney(stayCost, stayCostCurrency) : null
  ]
    .filter(Boolean)
    .join(" · ");
  const visibleTravelers = travelers.slice(0, 4);

  return (
    <aside className="flex flex-col gap-6 lg:sticky lg:top-[84px]">
      <SectionNav navigationGroups={navigationGroups} tripId={tripId} />

      <BudgetSummaryCard
        summary={budgetSummary}
        currency={budgetCurrency}
        isLoading={budgetLoading}
        canEdit={canMutateTrip}
        optimizationDisabled={optimizationDisabled}
        perPersonAverage={perPersonAverage}
        onOpenBudgetOptimization={onOpenBudgetOptimization}
      />

      {accommodation ? (
        <div className="rounded-[18px] border border-sand-300 bg-white p-5">
          <h2 className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            Stay
          </h2>
          <p className="mt-3 text-[14.5px] font-semibold text-cocoa-900">{accommodation.name}</p>
          {stayMeta ? (
            <p className="mt-1 text-[13px] leading-[1.5] text-cocoa-400">{stayMeta}</p>
          ) : null}
        </div>
      ) : null}

      <div className="rounded-[18px] border border-sand-300 bg-white p-5">
        <h2 className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
          Traveling with
        </h2>
        <div className="mt-3 flex items-center gap-2.5">
          {visibleTravelers.length > 0 ? (
            <div className="flex">
              {visibleTravelers.map((traveler, index) => (
                <span
                  key={traveler.id}
                  title={traveler.name}
                  className={`flex h-8 w-8 items-center justify-center rounded-full border-2 border-white text-[12px] font-semibold ${
                    AVATAR_COLORS[index % AVATAR_COLORS.length]
                  } ${index > 0 ? "-ml-2" : ""}`}
                >
                  {initials(traveler.name)}
                </span>
              ))}
            </div>
          ) : (
            <span className="text-[13px] text-cocoa-400">Just you so far</span>
          )}
          <a
            href="#sharing"
            aria-label="Invite collaborator"
            className="flex h-8 w-8 items-center justify-center rounded-full border border-dashed border-sand-600 text-[#A08D78] transition hover:border-clay hover:text-clay"
          >
            <PlusIcon className="h-3.5 w-3.5" />
          </a>
        </div>
      </div>
    </aside>
  );
}

function formatStayDates(
  checkIn: string | null | undefined,
  checkOut: string | null | undefined
): string | null {
  if (!checkIn) {
    return null;
  }
  const inLabel = formatDate(checkIn, { month: "short", day: "numeric" });
  if (!checkOut) {
    return inLabel;
  }
  const outLabel = formatDate(checkOut, { month: "short", day: "numeric" });
  return `${inLabel} – ${outLabel}`;
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}
