import { Button } from "@/shared/ui/button";

type PartialItineraryWarningProps = {
  missingDayCount: number;
  canEdit: boolean;
  onRegenerate: () => void;
  onEdit: () => void;
};

export function PartialItineraryWarning({
  missingDayCount,
  canEdit,
  onRegenerate,
  onEdit
}: PartialItineraryWarningProps) {
  return (
    <section className="rounded-[16px] border border-[#EAD9B8] bg-[#FDF7E8] p-4 text-[#7A5727]">
      <h2 className="font-semibold text-cocoa-900">This itinerary looks incomplete</h2>
      <p className="mt-1 text-[13.5px] leading-5">
        {missingDayCount > 0
          ? `${missingDayCount} day${missingDayCount === 1 ? " is" : "s are"} missing activities or a generated plan.`
          : "Some itinerary details are missing."}
      </p>
      <div className="mt-3 flex flex-wrap gap-2">
        <Button onClick={onRegenerate} size="sm" type="button">Regenerate missing day</Button>
        {canEdit ? <Button onClick={onEdit} size="sm" type="button" variant="secondary">Edit manually</Button> : null}
      </div>
    </section>
  );
}
