import { apiFetch } from "@/shared/api/client";
import type {
  CommentCount,
  CreateCommentRequest,
  ItineraryComment,
  UpdateCommentRequest
} from "@/entities/comment/model";

type ListCommentsResponse = {
  items: ItineraryComment[];
};

type CommentCountsResponse = {
  items: CommentCount[];
};

// React Query keys for itinerary comments. Comments are a private,
// authenticated collaboration feature and are never fetched from the public
// share page.
export const commentKeys = {
  all: (tripId: string) => ["trips", "detail", tripId, "comments"] as const,
  counts: (tripId: string) => [...commentKeys.all(tripId), "counts"] as const,
  item: (tripId: string, dayNumber: number, itemIndex: number) =>
    [...commentKeys.all(tripId), "item", dayNumber, itemIndex] as const
};

export async function listTripComments(tripId: string): Promise<ItineraryComment[]> {
  const response = await apiFetch<ListCommentsResponse>(`/trips/${tripId}/comments`);
  return response?.items ?? [];
}

export async function listItemComments(
  tripId: string,
  dayNumber: number,
  itemIndex: number
): Promise<ItineraryComment[]> {
  const query = new URLSearchParams({
    dayNumber: String(dayNumber),
    itemIndex: String(itemIndex)
  });
  const response = await apiFetch<ListCommentsResponse>(
    `/trips/${tripId}/comments?${query.toString()}`
  );
  return response?.items ?? [];
}

export async function listTripCommentCounts(tripId: string): Promise<CommentCount[]> {
  const response = await apiFetch<CommentCountsResponse>(`/trips/${tripId}/comments/counts`);
  return response?.items ?? [];
}

export function createItineraryComment(
  tripId: string,
  input: CreateCommentRequest
): Promise<ItineraryComment> {
  return apiFetch<ItineraryComment>(`/trips/${tripId}/comments`, {
    method: "POST",
    body: JSON.stringify({
      dayNumber: input.dayNumber,
      itemIndex: input.itemIndex,
      body: input.body.trim()
    })
  });
}

export function updateItineraryComment(
  tripId: string,
  commentId: string,
  input: UpdateCommentRequest
): Promise<ItineraryComment> {
  return apiFetch<ItineraryComment>(`/trips/${tripId}/comments/${commentId}`, {
    method: "PATCH",
    body: JSON.stringify({ body: input.body.trim() })
  });
}

export function deleteItineraryComment(
  tripId: string,
  commentId: string
): Promise<{ success: boolean }> {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/comments/${commentId}`, {
    method: "DELETE"
  });
}
