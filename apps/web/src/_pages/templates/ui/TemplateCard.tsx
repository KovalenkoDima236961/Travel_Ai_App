import Link from "next/link";
import { formatBudget } from "@/lib/utils";
import type { TripTemplate } from "@/entities/trip-template/model";
import { CalendarIcon, CurrencyIcon, RectangleStackIcon } from "./icons";

// Private = warm clay tint, Workspace = muted green. The green is a one-off
// (not in the token palette) so it stays as an arbitrary value.
const BADGE = {
  private: "bg-clay-tint text-clay-deep",
  workspace: "bg-[#E7EEE9] text-[#3E6B5A]"
} as const;

type TemplateCardProps = {
  template: TripTemplate;
};

/**
 * Slice-local template card for the redesigned Templates screen. The whole card
 * is a link into the template detail page (where Use / Adapt / Duplicate /
 * Archive live), mirroring the redesigned TripCard's link-only pattern.
 */
export function TemplateCard({ template }: TemplateCardProps) {
  const duration = `${template.durationDays} ${template.durationDays === 1 ? "day" : "days"}`;
  const estimate = formatBudget(
    template.estimatedTotalAmount,
    template.estimatedTotalCurrency || template.defaultCurrency || "EUR"
  );

  return (
    <Link
      href={`/templates/${template.id}`}
      className="group flex flex-col rounded-[18px] border border-sand-300 bg-white px-6 py-[22px] shadow-[0_1px_2px_rgba(34,26,20,0.04)] transition duration-200 hover:-translate-y-[3px] hover:shadow-[0_18px_40px_rgba(34,26,20,0.1)]"
    >
      <div className="flex items-start justify-between gap-3">
        <span className="flex h-11 w-11 items-center justify-center rounded-[13px] bg-clay-tint text-clay-dark">
          <RectangleStackIcon className="h-[21px] w-[21px]" />
        </span>
        <span
          className={`inline-flex items-center rounded-full px-[11px] py-1 text-[11.5px] font-semibold capitalize ${BADGE[template.visibility]}`}
        >
          {template.visibility}
        </span>
      </div>

      <h2 className="mt-4 font-newsreader text-[22px] font-semibold tracking-[-0.01em] text-cocoa-900">
        {template.title}
      </h2>
      <p className="mt-2 flex-1 text-[13.5px] leading-[1.55] text-cocoa-500">
        {template.description || template.destinationHint || "Reusable itinerary structure."}
      </p>

      <div className="mt-4 flex items-center gap-3.5 border-t border-sand-200 pt-3.5 text-[13px] text-cocoa-400">
        <span className="inline-flex items-center gap-1.5">
          <CalendarIcon className="h-3.5 w-3.5 text-[#B09E8A]" />
          {duration}
        </span>
        <span className="inline-flex items-center gap-1.5">
          <CurrencyIcon className="h-3.5 w-3.5 text-[#B09E8A]" />
          {estimate}
        </span>
      </div>
    </Link>
  );
}
