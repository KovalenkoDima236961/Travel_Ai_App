"use client";

import { FormEvent, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import {
  acceptTripSuggestion,
  createTripSuggestion,
  listTripSuggestions,
  listTripVotes,
  rejectTripSuggestion,
  resolveTripSuggestion,
  setTripVote,
  tripKeys
} from "@/lib/api/trips";
import { activityKeys } from "@/lib/api/activity";
import { formatDate, getErrorMessage } from "@/lib/utils";
import type {
  TripSuggestion,
  TripSuggestionStatus,
  TripVoteTargetType,
  TripVoteType
} from "@/entities/collaboration/model";

type TripPlanningCollaborationPanelProps = {
  tripId: string;
  canSuggest: boolean;
  canResolve: boolean;
  expectedItineraryRevision?: number;
};

export function TripPlanningCollaborationPanel({
  tripId,
  canSuggest,
  canResolve,
  expectedItineraryRevision
}: TripPlanningCollaborationPanelProps) {
  const queryClient = useQueryClient();
  const [suggestionComment, setSuggestionComment] = useState("");
  const [voteTargetType, setVoteTargetType] = useState<TripVoteTargetType>("activity");
  const [voteTargetId, setVoteTargetId] = useState("");
  const [voteType, setVoteType] = useState<TripVoteType>("thumbs_up");
  const [statusFilter, setStatusFilter] = useState<TripSuggestionStatus | "all">("open");
  const [error, setError] = useState<string | null>(null);

  const suggestionsQuery = useQuery({
    queryKey: tripKeys.suggestions(tripId, statusFilter === "all" ? undefined : statusFilter),
    queryFn: () => listTripSuggestions(tripId, statusFilter === "all" ? undefined : statusFilter),
    enabled: canSuggest && Boolean(tripId)
  });

  const votesQuery = useQuery({
    queryKey: tripKeys.votes(tripId),
    queryFn: () => listTripVotes(tripId),
    enabled: canSuggest && Boolean(tripId)
  });

  async function refreshPlanning() {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [...tripKeys.detail(tripId), "suggestions"] }),
      queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
      queryClient.invalidateQueries({ queryKey: [...tripKeys.detail(tripId), "votes"] }),
      queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
    ]);
  }

  const createSuggestionMutation = useMutation({
    mutationFn: () =>
      createTripSuggestion(tripId, {
        suggestionType: "note",
        targetType: "trip",
        comment: suggestionComment
      }),
    onSuccess: async () => {
      setSuggestionComment("");
      setError(null);
      await refreshPlanning();
    },
    onError: (err) => setError(getErrorMessage(err, "Could not create suggestion."))
  });

  const updateSuggestionMutation = useMutation({
    mutationFn: ({ suggestion, action }: { suggestion: TripSuggestion; action: "accept" | "reject" | "resolve" }) => {
      if (action === "accept") {
        return acceptTripSuggestion(tripId, suggestion.id, expectedItineraryRevision);
      }
      if (action === "reject") {
        return rejectTripSuggestion(tripId, suggestion.id);
      }
      return resolveTripSuggestion(tripId, suggestion.id);
    },
    onSuccess: async () => {
      setError(null);
      await refreshPlanning();
    },
    onError: (err) => setError(getErrorMessage(err, "Could not update suggestion."))
  });

  const voteMutation = useMutation({
    mutationFn: () =>
      setTripVote(tripId, {
        targetType: voteTargetType,
        targetId: voteTargetId,
        voteType
      }),
    onSuccess: async () => {
      setVoteTargetId("");
      setError(null);
      await refreshPlanning();
    },
    onError: (err) => setError(getErrorMessage(err, "Could not save vote."))
  });

  if (!canSuggest) {
    return null;
  }

  const suggestions = suggestionsQuery.data ?? [];
  const voteSummaries = votesQuery.data ?? [];
  const busy =
    createSuggestionMutation.isPending ||
    updateSuggestionMutation.isPending ||
    voteMutation.isPending;

  function submitSuggestion(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!suggestionComment.trim()) {
      setError("Enter a suggestion.");
      return;
    }
    createSuggestionMutation.mutate();
  }

  function submitVote(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!voteTargetId.trim()) {
      setError("Enter a vote target.");
      return;
    }
    voteMutation.mutate();
  }

  return (
    <Card>
      <div className="flex flex-col gap-5">
        <div>
          <h2 className="text-lg font-semibold text-slate-950">Planning decisions</h2>
          <p className="mt-2 text-sm leading-6 text-slate-600">
            Capture proposed changes and quick votes without editing the itinerary first.
          </p>
        </div>

        {error ? (
          <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
            {error}
          </div>
        ) : null}

        <form className="space-y-3" onSubmit={submitSuggestion}>
          <label className="block text-sm font-medium text-slate-700" htmlFor="trip-suggestion">
            Suggestion
          </label>
          <Textarea
            disabled={busy}
            id="trip-suggestion"
            maxLength={2000}
            onChange={(event) => setSuggestionComment(event.target.value)}
            placeholder="Suggest a change to the trip plan"
            value={suggestionComment}
          />
          <div className="flex justify-end">
            <Button disabled={busy || !suggestionComment.trim()} type="submit">
              {createSuggestionMutation.isPending ? "Saving..." : "Add suggestion"}
            </Button>
          </div>
        </form>

        <div className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <h3 className="text-sm font-semibold text-slate-900">Suggestions</h3>
            <Select
              className="max-w-[160px]"
              onChange={(event) => setStatusFilter(event.target.value as TripSuggestionStatus | "all")}
              value={statusFilter}
            >
              <option value="open">Open</option>
              <option value="accepted">Accepted</option>
              <option value="rejected">Rejected</option>
              <option value="resolved">Resolved</option>
              <option value="all">All</option>
            </Select>
          </div>
          {suggestionsQuery.isPending ? (
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
              Loading suggestions...
            </div>
          ) : null}
          {suggestions.length === 0 && !suggestionsQuery.isPending ? (
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
              No suggestions in this view.
            </div>
          ) : null}
          {suggestions.map((suggestion) => (
            <div className="rounded-lg border border-slate-200 bg-white p-3" key={suggestion.id}>
              <div className="flex flex-col gap-3">
                <div>
                  <p className="text-sm font-semibold text-slate-950">
                    {labelForSuggestion(suggestion)}
                  </p>
                  <p className="mt-1 text-xs text-slate-500">
                    {suggestion.status} · {formatDate(suggestion.createdAt, { dateStyle: "medium" })}
                  </p>
                  {suggestion.comment ? (
                    <p className="mt-2 whitespace-pre-wrap text-sm leading-6 text-slate-700">
                      {suggestion.comment}
                    </p>
                  ) : null}
                </div>
                {canResolve && suggestion.status === "open" ? (
                  <div className="flex flex-wrap gap-2">
                    <Button
                      disabled={busy}
                      onClick={() => updateSuggestionMutation.mutate({ suggestion, action: "accept" })}
                      size="sm"
                      type="button"
                    >
                      Accept
                    </Button>
                    <Button
                      disabled={busy}
                      onClick={() => updateSuggestionMutation.mutate({ suggestion, action: "reject" })}
                      size="sm"
                      type="button"
                      variant="secondary"
                    >
                      Reject
                    </Button>
                    <Button
                      disabled={busy}
                      onClick={() => updateSuggestionMutation.mutate({ suggestion, action: "resolve" })}
                      size="sm"
                      type="button"
                      variant="secondary"
                    >
                      Resolve
                    </Button>
                  </div>
                ) : null}
              </div>
            </div>
          ))}
        </div>

        <form className="space-y-3" onSubmit={submitVote}>
          <h3 className="text-sm font-semibold text-slate-900">Quick vote</h3>
          <div className="grid gap-3 sm:grid-cols-2">
            <Select
              disabled={busy}
              onChange={(event) => setVoteTargetType(event.target.value as TripVoteTargetType)}
              value={voteTargetType}
            >
              <option value="activity">Activity</option>
              <option value="restaurant">Restaurant</option>
              <option value="hotel">Hotel</option>
              <option value="destination">Destination</option>
              <option value="suggestion">Suggestion</option>
            </Select>
            <Select
              disabled={busy}
              onChange={(event) => setVoteType(event.target.value as TripVoteType)}
              value={voteType}
            >
              <option value="thumbs_up">Thumbs up</option>
              <option value="thumbs_down">Thumbs down</option>
              <option value="heart">Heart</option>
              <option value="star">Star</option>
            </Select>
          </div>
          <Input
            disabled={busy}
            onChange={(event) => setVoteTargetId(event.target.value)}
            placeholder="Target ID, such as 1:0 or a suggestion ID"
            value={voteTargetId}
          />
          <div className="flex justify-end">
            <Button disabled={busy || !voteTargetId.trim()} type="submit">
              {voteMutation.isPending ? "Saving..." : "Vote"}
            </Button>
          </div>
        </form>

        {voteSummaries.length > 0 ? (
          <div className="space-y-3">
            <h3 className="text-sm font-semibold text-slate-900">Vote summary</h3>
            {voteSummaries.map((summary) => (
              <div
                className="rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-700"
                key={`${summary.targetType}:${summary.targetId}`}
              >
                <p className="font-medium text-slate-900">
                  {summary.targetType} · {summary.targetId}
                </p>
                <p className="mt-1 text-xs text-slate-500">
                  {formatVoteCounts(summary.counts)}
                  {summary.currentVote ? ` · your vote: ${summary.currentVote}` : ""}
                </p>
              </div>
            ))}
          </div>
        ) : null}
      </div>
    </Card>
  );
}

function labelForSuggestion(suggestion: TripSuggestion) {
  const target = suggestion.targetId ? ` · ${suggestion.targetId}` : "";
  return `${suggestion.suggestionType.replaceAll("_", " ")} · ${suggestion.targetType}${target}`;
}

function formatVoteCounts(counts: Partial<Record<TripVoteType, number>>) {
  const labels: Array<[TripVoteType, string]> = [
    ["thumbs_up", "thumbs up"],
    ["thumbs_down", "thumbs down"],
    ["heart", "heart"],
    ["star", "star"]
  ];
  const parts = labels
    .map(([key, label]) => `${label}: ${counts[key] ?? 0}`)
    .filter((part) => !part.endsWith(": 0"));
  return parts.length > 0 ? parts.join(" · ") : "No votes";
}
