"use client";

import { PollResultsBar } from "./PollResultsBar";
import { PollStatusBadge } from "./PollStatusBadge";
import { PollVoteControls } from "./PollVoteControls";
import type { TripPoll, VoteTripPollInput } from "@/types/trip-decisions";

type PollCardProps = {
  poll: TripPoll;
  disabled?: boolean;
  isVoting?: boolean;
  isClosing?: boolean;
  isArchiving?: boolean;
  onVote: (pollId: string, input: VoteTripPollInput) => void;
  onClose: (pollId: string) => void;
  onArchive: (pollId: string) => void;
};

export function PollCard({
  poll,
  disabled = false,
  isVoting = false,
  isClosing = false,
  isArchiving = false,
  onVote,
  onClose,
  onArchive
}: PollCardProps) {
  const hasVotes = poll.results.totalVotes > 0;

  return (
    <article className="rounded-[16px] border border-sand-300 bg-white p-4">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-[15px] font-semibold text-cocoa-900">{poll.title}</h3>
            <PollStatusBadge status={poll.status} />
          </div>
          {poll.description ? (
            <p className="mt-1 text-[13px] leading-5 text-cocoa-500">{poll.description}</p>
          ) : null}
        </div>
        {poll.canManage ? (
          <div className="flex shrink-0 gap-1">
            {poll.status === "open" ? (
              <button
                className="rounded-full px-2.5 py-1 text-[12px] font-semibold text-cocoa-500 transition hover:bg-sand-100 hover:text-cocoa-900 disabled:opacity-50"
                disabled={disabled || isClosing}
                onClick={() => onClose(poll.id)}
                type="button"
              >
                Close
              </button>
            ) : null}
            <button
              className="rounded-full px-2.5 py-1 text-[12px] font-semibold text-cocoa-500 transition hover:bg-sand-100 hover:text-cocoa-900 disabled:opacity-50"
              disabled={disabled || isArchiving}
              onClick={() => onArchive(poll.id)}
              type="button"
            >
              Archive
            </button>
          </div>
        ) : null}
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1fr)]">
        <PollVoteControls
          disabled={disabled}
          isPending={isVoting}
          onVote={(input) => onVote(poll.id, input)}
          poll={poll}
        />
        <div className="space-y-3 rounded-[14px] bg-sand-50 p-3">
          {hasVotes ? (
            poll.results.options.map((result) => (
              <PollResultsBar key={result.optionId} result={result} />
            ))
          ) : (
            <p className="text-[13px] text-cocoa-400">No votes yet.</p>
          )}
        </div>
      </div>
    </article>
  );
}
