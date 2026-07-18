import { GenerationQualityBadges } from "@/components/generation-quality";
import type { BudgetSummary } from "@/entities/budget/model";
import type { GenerationQuality } from "@/types/generation-quality";
import type { BudgetConfidence } from "@/types/budget-confidence";
import type { TripHealth } from "@/types/trip-health";
import type { RealWorldReadiness } from "@/types/verification";
import type { Trip } from "@/entities/trip/model";

type PostGenerationReviewPanelProps = {
  trip: Trip;
  quality?: GenerationQuality | null;
  budgetSummary?: BudgetSummary | null;
  budgetConfidence?: BudgetConfidence | null;
  health?: TripHealth | null;
  verification?: RealWorldReadiness | null;
  onOpenItinerary: () => void;
};

export function PostGenerationReviewPanel({
  trip,
  quality,
  budgetSummary,
  budgetConfidence,
  health,
  verification,
  onOpenItinerary
}: PostGenerationReviewPanelProps) {
  const itemCount = trip.itinerary?.days.reduce((count, day) => count + day.items.length, 0) ?? 0;
  const routeWarnings = health?.issues.filter(
    (issue) => issue.status === "open" && (issue.category === "route" || issue.category === "transport")
  ).length;
  const availabilityWarnings = verification
    ? verification.summary.needsReviewCount + verification.summary.missingCount + verification.summary.staleCount
    : null;
  const nextAction = nextBestAction(trip, health, verification, budgetConfidence);
  const reviewNotes = reviewReasons(quality, health, verification, budgetSummary);

  return (
    <section className="rounded-[18px] border border-[#DCE8DD] bg-white p-5 shadow-[0_1px_2px_rgba(34,26,20,0.04)]">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="font-newsreader text-[23px] font-semibold text-cocoa-900">Your itinerary is ready</h2>
            <GenerationQualityBadges quality={quality} source={trip.itinerary?.source} />
          </div>
          <p className="mt-1.5 text-[14px] leading-6 text-cocoa-500">
            Start with the key checks below, then open the day-by-day itinerary.
          </p>
        </div>
        <button
          className="inline-flex h-10 items-center justify-center rounded-full bg-clay px-4 text-[13.5px] font-semibold text-sand-100 transition hover:bg-clay-dark"
          onClick={onOpenItinerary}
          type="button"
        >
          Open itinerary
        </button>
      </div>

      <dl className="mt-5 grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Metric label="Generated" value={`${trip.itinerary?.days.length ?? 0} days · ${itemCount} items`} />
        <Metric label="Budget confidence" value={budgetConfidence ? formatLevel(budgetConfidence.level) : "Not checked yet"} />
        <Metric label="Real-world readiness" value={verification ? formatLevel(verification.level) : "Not checked yet"} />
        <Metric label="Trip health" value={health ? formatLevel(health.level) : "Not checked yet"} />
        <Metric label="Route warnings" value={routeWarnings != null ? String(routeWarnings) : "Not checked yet"} />
        <Metric label="Missing prices" value={budgetSummary ? String(budgetSummary.missingEstimateCount) : "Not checked yet"} />
        <Metric label="Availability warnings" value={availabilityWarnings != null ? String(availabilityWarnings) : "Not checked yet"} />
        <Metric label="AI checks" value={quality ? formatQuality(quality) : "Status unavailable"} />
      </dl>

      <div className="mt-5 grid gap-3 lg:grid-cols-[minmax(0,1fr)_260px]">
        <div className="rounded-xl bg-sand-50 p-4">
          <p className="text-[13px] font-semibold text-cocoa-800">Why it may need review</p>
          <ul className="mt-2 space-y-1.5 text-[13px] leading-5 text-cocoa-600">
            {reviewNotes.map((note) => <li key={note}>• {note}</li>)}
          </ul>
        </div>
        <a
          className="rounded-xl border border-clay/30 bg-clay-tint/50 p-4 transition hover:border-clay"
          href={nextAction.href}
        >
          <p className="text-[12px] font-semibold uppercase tracking-[0.06em] text-clay-deep">Recommended next</p>
          <p className="mt-1 text-[14px] font-semibold text-cocoa-900">{nextAction.label}</p>
          <p className="mt-1 text-[12.5px] leading-5 text-cocoa-500">{nextAction.reason}</p>
        </a>
      </div>
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-sand-300 bg-sand-50 px-3.5 py-3">
      <dt className="text-[11.5px] font-semibold uppercase tracking-[0.06em] text-[#A08D78]">{label}</dt>
      <dd className="mt-1 text-[13px] font-medium leading-5 text-cocoa-800">{value}</dd>
    </div>
  );
}

function formatLevel(value: string) {
  return value.replaceAll("_", " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function formatQuality(quality: GenerationQuality) {
  if (quality.status === "repaired_and_validated" || quality.status === "repaired_with_warnings") return "Repaired";
  if (quality.status === "validated" || quality.status === "validated_with_warnings") return "Validated";
  return "Needs review";
}

function reviewReasons(
  quality: GenerationQuality | null | undefined,
  health: TripHealth | null | undefined,
  verification: RealWorldReadiness | null | undefined,
  budget: BudgetSummary | null | undefined
) {
  const reasons = [
    ...(quality?.remainingIssues.slice(0, 2).map((issue) => issue.description || issue.title) ?? []),
    ...(health?.issues.filter((issue) => issue.status === "open").slice(0, 2).map((issue) => issue.description || issue.title) ?? [])
  ];
  if (budget && budget.missingEstimateCount > 0) reasons.push(`${budget.missingEstimateCount} price estimate${budget.missingEstimateCount === 1 ? " is" : "s are"} missing.`);
  if (verification && verification.summary.missingCount > 0) reasons.push(`${verification.summary.missingCount} detail${verification.summary.missingCount === 1 ? " has" : "s have"} missing real-world verification.`);
  return reasons.length > 0 ? reasons.slice(0, 3) : ["Your AI draft passed the available app consistency checks. Confirm real-world details before booking."];
}

function nextBestAction(
  trip: Trip,
  health: TripHealth | null | undefined,
  verification: RealWorldReadiness | null | undefined,
  budget: BudgetConfidence | null | undefined
) {
  const healthFix = health?.topFixes[0];
  if (healthFix) return { label: healthFix.label, href: healthFix.href, reason: "This is the highest-priority trip health issue." };
  const verificationAction = verification?.recommendedActions[0];
  if (verificationAction) return { label: verificationAction.label, href: verificationAction.href, reason: "This improves real-world readiness." };
  if (budget && ["very_low", "low"].includes(budget.level)) return { label: "Review budget", href: "#budget", reason: "Some costs need a more reliable estimate." };
  if (!trip.accommodation) return { label: "Add accommodation", href: "#accommodation", reason: "A stay helps improve route and budget checks." };
  return { label: "Open itinerary", href: "#itinerary", reason: "Review the day-by-day plan and adjust anything that does not fit." };
}
