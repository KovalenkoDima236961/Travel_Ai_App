"use client";

import { useState } from "react";
import { CreatePollDialog } from "./CreatePollDialog";
import { PollCard } from "./PollCard";
import { Button } from "@/shared/ui/button";
import { useArchiveTripPoll } from "@/hooks/useArchiveTripPoll";
import { useCloseTripPoll } from "@/hooks/useCloseTripPoll";
import { useCreateTripPoll } from "@/hooks/useCreateTripPoll";
import { useTripPolls } from "@/hooks/useTripPolls";
import { useVoteTripPoll } from "@/hooks/useVoteTripPoll";
import { getErrorMessage } from "@/lib/utils";
import type { CreateTripPollInput, VoteTripPollInput } from "@/types/trip-decisions";

type PollsPanelProps = {
  tripId: string;
  canCreate: boolean;
  online: boolean;
};

export function PollsPanel({ tripId, canCreate, online }: PollsPanelProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pollsQuery = useTripPolls(tripId, online);
  const createMutation = useCreateTripPoll(tripId);
  const voteMutation = useVoteTripPoll(tripId);
  const closeMutation = useCloseTripPoll(tripId);
  const archiveMutation = useArchiveTripPoll(tripId);
  const polls = pollsQuery.data ?? [];
  const openPolls = polls.filter((poll) => poll.status === "open");
  const closedPolls = polls.filter((poll) => poll.status !== "open");

  async function createPoll(input: CreateTripPollInput) {
    try {
      setError(null);
      await createMutation.mutateAsync(input);
      setDialogOpen(false);
    } catch (err) {
      setError(getErrorMessage(err, "Could not create poll."));
    }
  }

  function vote(pollId: string, input: VoteTripPollInput) {
    voteMutation.mutate({ pollId, input });
  }

  return (
    <section id="decisions" className="scroll-mt-24 rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 className="font-newsreader text-[25px] font-semibold text-cocoa-900">
            Decisions
          </h2>
        </div>
        {canCreate ? (
          <Button disabled={!online} onClick={() => setDialogOpen(true)} size="sm" type="button">
            Create poll
          </Button>
        ) : null}
      </div>

      {!online ? (
        <p className="mt-4 rounded-[12px] bg-sand-100 px-3 py-2 text-[13px] text-cocoa-500">
          You need to be online to vote.
        </p>
      ) : null}

      <div className="mt-5 space-y-4">
        {pollsQuery.isLoading ? (
          <p className="text-[13px] text-cocoa-400">Loading polls...</p>
        ) : null}
        {openPolls.map((poll) => (
          <PollCard
            key={poll.id}
            disabled={!online}
            isArchiving={archiveMutation.isPending}
            isClosing={closeMutation.isPending}
            isVoting={voteMutation.isPending}
            onArchive={(pollId) => archiveMutation.mutate(pollId)}
            onClose={(pollId) => closeMutation.mutate(pollId)}
            onVote={vote}
            poll={poll}
          />
        ))}
        {closedPolls.length > 0 ? (
          <div className="space-y-3">
            <h3 className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              Closed
            </h3>
            {closedPolls.map((poll) => (
              <PollCard
                key={poll.id}
                disabled={!online}
                isArchiving={archiveMutation.isPending}
                isClosing={closeMutation.isPending}
                isVoting={voteMutation.isPending}
                onArchive={(pollId) => archiveMutation.mutate(pollId)}
                onClose={(pollId) => closeMutation.mutate(pollId)}
                onVote={vote}
                poll={poll}
              />
            ))}
          </div>
        ) : null}
        {!pollsQuery.isLoading && polls.length === 0 ? (
          <p className="rounded-[14px] bg-sand-50 p-4 text-[13px] text-cocoa-500">
            No decisions yet.
          </p>
        ) : null}
      </div>

      <CreatePollDialog
        error={error}
        isPending={createMutation.isPending}
        onCreate={createPoll}
        onOpenChange={setDialogOpen}
        open={dialogOpen}
      />
    </section>
  );
}
