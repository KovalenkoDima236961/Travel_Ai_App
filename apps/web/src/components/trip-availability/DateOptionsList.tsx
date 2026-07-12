"use client";

import { useEffect, useState } from "react";
import { DateOptionCard } from "./DateOptionCard";
import { Button } from "@/shared/ui/button";
import type { DateOptionsResult, TripDateOption } from "@/types/trip-availability";

type DateOptionsListProps = {
  canEdit: boolean;
  disabled?: boolean;
  isApplying?: boolean;
  isCreatingPoll?: boolean;
  result?: DateOptionsResult;
  onApply: (option: TripDateOption) => void;
  onCreatePoll: (optionIds: string[]) => void;
};

export function DateOptionsList({
  canEdit,
  disabled = false,
  isApplying = false,
  isCreatingPoll = false,
  result,
  onApply,
  onCreatePoll
}: DateOptionsListProps) {
  const options = result?.options ?? [];
  const recommendedID = result?.summary.recommendedOptionId;
  const [selectedOptionIds, setSelectedOptionIds] = useState<string[]>([]);

  useEffect(() => {
    setSelectedOptionIds(options.slice(0, 3).map((option) => option.id));
  }, [options]);

  function toggle(optionId: string, checked: boolean) {
    setSelectedOptionIds((current) =>
      checked ? [...new Set([...current, optionId])] : current.filter((id) => id !== optionId)
    );
  }

  if (!result) {
    return <p className="text-[13px] text-cocoa-400">Loading date options...</p>;
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-[13px] text-cocoa-500">
          {result.summary.responseCount} of {result.summary.totalCollaborators} travelers submitted
          availability.
        </p>
        {canEdit ? (
          <Button
            disabled={disabled || selectedOptionIds.length === 0 || isCreatingPoll}
            onClick={() => onCreatePoll(selectedOptionIds)}
            size="sm"
            type="button"
            variant="secondary"
          >
            {isCreatingPoll ? "Creating poll..." : "Create poll"}
          </Button>
        ) : null}
      </div>

      {options.length > 0 ? (
        <div className="space-y-3">
          {options.map((option) => (
            <DateOptionCard
              key={option.id}
              canApply={canEdit}
              checked={selectedOptionIds.includes(option.id)}
              disabled={disabled || isApplying}
              isRecommended={option.id === recommendedID}
              onApply={onApply}
              onCheckedChange={(checked) => toggle(option.id, checked)}
              option={option}
            />
          ))}
        </div>
      ) : (
        <p className="rounded-[14px] bg-sand-50 p-4 text-[13px] text-cocoa-500">
          No viable date options yet. Add availability or widen the search window.
        </p>
      )}
    </div>
  );
}
