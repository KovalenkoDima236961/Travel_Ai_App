"use client";

import { useMemo, useState } from "react";
import { Button } from "@/shared/ui/button";
import type { TripPoll, VoteTripPollInput } from "@/types/trip-decisions";

type PollVoteControlsProps = {
  poll: TripPoll;
  disabled?: boolean;
  isPending?: boolean;
  onVote: (input: VoteTripPollInput) => void;
};

export function PollVoteControls({
  poll,
  disabled = false,
  isPending = false,
  onVote
}: PollVoteControlsProps) {
  const currentOptionIds = useMemo(
    () => new Set(poll.userVotes.map((vote) => vote.optionId).filter(Boolean) as string[]),
    [poll.userVotes]
  );
  const [selected, setSelected] = useState<string[]>(Array.from(currentOptionIds));
  const [rating, setRating] = useState<number>(poll.userVotes[0]?.ratingValue ?? 3);

  const isMultiple = poll.pollType === "multiple_choice";
  const isRating = poll.pollType === "rating";
  const canSubmit = poll.canVote && !disabled && !isPending;

  function toggle(optionId: string) {
    setSelected((current) => {
      if (!isMultiple) {
        return [optionId];
      }
      return current.includes(optionId)
        ? current.filter((id) => id !== optionId)
        : [...current, optionId];
    });
  }

  function submit() {
    if (!canSubmit) {
      return;
    }
    onVote({
      optionIds: selected,
      ratingValue: isRating ? rating : null
    });
  }

  if (!poll.canVote) {
    return (
      <p className="rounded-[12px] bg-sand-100 px-3 py-2 text-[13px] text-cocoa-500">
        Voting is closed.
      </p>
    );
  }

  return (
    <div className="space-y-3">
      <div className="grid gap-2">
        {poll.options.map((option) => {
          const active = selected.includes(option.id);
          return (
            <label
              key={option.id}
              className={`flex cursor-pointer items-start gap-2 rounded-[12px] border px-3 py-2 text-[13px] transition ${
                active
                  ? "border-clay bg-[#FBF0EB] text-cocoa-900"
                  : "border-sand-300 bg-white text-cocoa-600 hover:border-sand-500"
              }`}
            >
              <input
                checked={active}
                className="mt-0.5"
                disabled={disabled || isPending}
                name={`poll-${poll.id}`}
                onChange={() => toggle(option.id)}
                type={isMultiple ? "checkbox" : "radio"}
              />
              <span>
                <span className="font-semibold">{option.label}</span>
                {option.description ? (
                  <span className="mt-0.5 block text-[12px] text-cocoa-400">
                    {option.description}
                  </span>
                ) : null}
              </span>
            </label>
          );
        })}
      </div>
      {isRating ? (
        <label className="block text-[13px] font-semibold text-cocoa-600">
          Rating
          <input
            className="mt-2 w-full"
            disabled={disabled || isPending}
            max={5}
            min={1}
            onChange={(event) => setRating(Number(event.target.value))}
            type="range"
            value={rating}
          />
          <span className="mt-1 block text-[12px] text-cocoa-400">{rating} out of 5</span>
        </label>
      ) : null}
      <Button
        disabled={!canSubmit || selected.length === 0}
        onClick={submit}
        size="sm"
        type="button"
      >
        {poll.userVotes.length > 0 ? "Change vote" : "Vote"}
      </Button>
    </div>
  );
}
