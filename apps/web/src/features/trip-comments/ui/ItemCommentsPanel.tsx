"use client";

import { FormEvent, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "@/shared/ui/button";
import { Textarea } from "@/shared/ui/textarea";
import {
  commentKeys,
  createItineraryComment,
  deleteItineraryComment,
  listItemComments,
  updateItineraryComment
} from "@/lib/api/comments";
import { activityKeys } from "@/lib/api/activity";
import { formatDate, getErrorMessage } from "@/lib/utils";
import type { ItineraryComment } from "@/entities/comment/model";

const MAX_COMMENT_LENGTH = 2000;

type ItemCommentsPanelProps = {
  tripId: string;
  dayNumber: number;
  itemIndex: number;
  itemTitle: string;
  itemTime?: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  currentUserId?: string;
  canComment: boolean;
};

export function ItemCommentsPanel({
  tripId,
  dayNumber,
  itemIndex,
  itemTitle,
  itemTime,
  open,
  onOpenChange,
  currentUserId,
  canComment
}: ItemCommentsPanelProps) {
  const queryClient = useQueryClient();
  const [body, setBody] = useState("");
  const [error, setError] = useState<string | null>(null);

  const commentsQuery = useQuery({
    queryKey: commentKeys.item(tripId, dayNumber, itemIndex),
    queryFn: () => listItemComments(tripId, dayNumber, itemIndex),
    enabled: open && Boolean(tripId)
  });

  async function refreshComments() {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: commentKeys.item(tripId, dayNumber, itemIndex)
      }),
      queryClient.invalidateQueries({ queryKey: commentKeys.counts(tripId) }),
      queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
    ]);
  }

  const createMutation = useMutation({
    mutationFn: (value: string) =>
      createItineraryComment(tripId, { dayNumber, itemIndex, body: value }),
    onSuccess: async () => {
      setBody("");
      setError(null);
      await refreshComments();
    },
    onError: (err) => setError(getErrorMessage(err, "Could not post comment."))
  });

  if (!open) {
    return null;
  }

  const comments = commentsQuery.data ?? [];
  const trimmed = body.trim();
  const tooLong = body.length > MAX_COMMENT_LENGTH;
  const canPost = canComment && trimmed.length > 0 && !tooLong && !createMutation.isPending;

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canPost) {
      return;
    }
    createMutation.mutate(trimmed);
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-950/40 px-4 py-10">
      <div
        aria-modal="true"
        className="w-full max-w-2xl rounded-lg bg-white p-5 shadow-xl"
        role="dialog"
      >
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <h2 className="text-lg font-semibold text-slate-950">Comments</h2>
            <p className="mt-1 truncate text-sm text-slate-600">
              Day {dayNumber}
              {itemTime ? ` · ${itemTime}` : ""} — {itemTitle}
            </p>
          </div>
          <Button onClick={() => onOpenChange(false)} type="button" variant="ghost">
            Close
          </Button>
        </div>

        <div className="mt-5 space-y-3">
          {commentsQuery.isPending ? (
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
              Loading comments...
            </div>
          ) : null}

          {commentsQuery.isError ? (
            <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
              {getErrorMessage(commentsQuery.error, "Could not load comments.")}
            </div>
          ) : null}

          {!commentsQuery.isPending && !commentsQuery.isError && comments.length === 0 ? (
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
              No comments yet.
            </div>
          ) : null}

          {comments.map((comment) => (
            <CommentRow
              comment={comment}
              currentUserId={currentUserId}
              dayNumber={dayNumber}
              itemIndex={itemIndex}
              key={comment.id}
              onChanged={refreshComments}
              tripId={tripId}
            />
          ))}
        </div>

        {error ? (
          <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
            {error}
          </div>
        ) : null}

        {canComment ? (
          <form className="mt-5 space-y-2" onSubmit={submit}>
            <label className="block text-sm font-medium text-slate-700" htmlFor="new-comment">
              Add a comment
            </label>
            <Textarea
              disabled={createMutation.isPending}
              id="new-comment"
              maxLength={MAX_COMMENT_LENGTH}
              onChange={(event) => setBody(event.target.value)}
              placeholder="Share feedback on this item..."
              value={body}
            />
            <div className="flex items-center justify-between">
              <span className={tooLong ? "text-xs text-red-600" : "text-xs text-slate-500"}>
                {body.length}/{MAX_COMMENT_LENGTH}
              </span>
              <Button disabled={!canPost} type="submit">
                {createMutation.isPending ? "Posting..." : "Post"}
              </Button>
            </div>
          </form>
        ) : (
          <p className="mt-5 text-sm text-slate-500">You do not have access to comment on this trip.</p>
        )}
      </div>
    </div>
  );
}

type CommentRowProps = {
  tripId: string;
  dayNumber: number;
  itemIndex: number;
  comment: ItineraryComment;
  currentUserId?: string;
  onChanged: () => Promise<void>;
};

function CommentRow({
  tripId,
  dayNumber,
  itemIndex,
  comment,
  currentUserId,
  onChanged
}: CommentRowProps) {
  const queryClient = useQueryClient();
  const [isEditing, setIsEditing] = useState(false);
  const [draft, setDraft] = useState(comment.body);
  const [error, setError] = useState<string | null>(null);

  const isAuthor = comment.isAuthor ?? (currentUserId ? comment.authorUserId === currentUserId : false);
  const authorLabel = isAuthor ? "You" : comment.authorDisplayName || "Collaborator";
  const edited = comment.updatedAt > comment.createdAt;

  const updateMutation = useMutation({
    mutationFn: (value: string) =>
      updateItineraryComment(tripId, comment.id, { body: value }),
    onSuccess: async () => {
      setIsEditing(false);
      setError(null);
      await onChanged();
    },
    onError: (err) => setError(getErrorMessage(err, "Could not update comment."))
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteItineraryComment(tripId, comment.id),
    onSuccess: async () => {
      setError(null);
      await onChanged();
    },
    onError: (err) => setError(getErrorMessage(err, "Could not delete comment."))
  });

  const busy = updateMutation.isPending || deleteMutation.isPending;

  function startEdit() {
    setDraft(comment.body);
    setError(null);
    setIsEditing(true);
  }

  function cancelEdit() {
    setIsEditing(false);
    setDraft(comment.body);
    setError(null);
  }

  function saveEdit() {
    const trimmed = draft.trim();
    if (trimmed.length === 0 || trimmed.length > MAX_COMMENT_LENGTH) {
      setError("Comment must be between 1 and 2000 characters.");
      return;
    }
    updateMutation.mutate(trimmed);
  }

  function remove() {
    if (!window.confirm("Delete this comment?")) {
      return;
    }
    deleteMutation.mutate();
    // Touch query client so a stale list is invalidated even if the panel
    // unmounts before onChanged resolves.
    void queryClient.invalidateQueries({
      queryKey: commentKeys.item(tripId, dayNumber, itemIndex)
    });
  }

  return (
    <div className="rounded-lg border border-slate-200 bg-white p-3">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-semibold text-slate-900">{authorLabel}</p>
        <p className="text-xs text-slate-500">
          {formatDate(comment.createdAt, { dateStyle: "medium", timeStyle: "short" })}
          {edited ? " · edited" : ""}
        </p>
      </div>

      {isEditing ? (
        <div className="mt-2 space-y-2">
          <Textarea
            disabled={busy}
            maxLength={MAX_COMMENT_LENGTH}
            onChange={(event) => setDraft(event.target.value)}
            value={draft}
          />
          <div className="flex items-center justify-between">
            <span className="text-xs text-slate-500">
              {draft.length}/{MAX_COMMENT_LENGTH}
            </span>
            <div className="flex gap-2">
              <Button disabled={busy} onClick={cancelEdit} size="sm" type="button" variant="secondary">
                Cancel
              </Button>
              <Button disabled={busy} onClick={saveEdit} size="sm" type="button">
                {updateMutation.isPending ? "Saving..." : "Save"}
              </Button>
            </div>
          </div>
        </div>
      ) : (
        <p className="mt-2 whitespace-pre-wrap text-sm leading-6 text-slate-700">{comment.body}</p>
      )}

      {error ? <p className="mt-2 text-xs text-red-600">{error}</p> : null}

      {!isEditing && (comment.canEdit || comment.canDelete) ? (
        <div className="mt-2 flex gap-3">
          {comment.canEdit ? (
            <button
              className="text-xs font-medium text-primary-700 hover:text-primary-600 disabled:opacity-60"
              disabled={busy}
              onClick={startEdit}
              type="button"
            >
              Edit
            </button>
          ) : null}
          {comment.canDelete ? (
            <button
              className="text-xs font-medium text-red-600 hover:text-red-700 disabled:opacity-60"
              disabled={busy}
              onClick={remove}
              type="button"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
