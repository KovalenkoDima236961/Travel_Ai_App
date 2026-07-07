"use client";

import type { CommentCount } from "@/entities/comment/model";

type TripCommentsSummaryProps = {
  counts: CommentCount[];
};

export function TripCommentsSummary({ counts }: TripCommentsSummaryProps) {
  const itemsWithComments = counts.filter((count) => count.count > 0);
  const totalComments = itemsWithComments.reduce((sum, count) => sum + count.count, 0);

  if (totalComments === 0) {
    return null;
  }

  const commentLabel = totalComments === 1 ? "comment" : "comments";
  const itemLabel = itemsWithComments.length === 1 ? "itinerary item" : "itinerary items";

  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4 text-sm text-slate-700">
      {totalComments} {commentLabel} across {itemsWithComments.length} {itemLabel}.
    </div>
  );
}
