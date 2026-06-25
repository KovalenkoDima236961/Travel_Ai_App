import { ApiError, apiFetch } from "@/lib/api/client";
import type {
  AcquireEditLockResponse,
  EditLockView,
  ReleaseEditLockResponse
} from "@/types/edit-locks";

type EditLockConflictPayload = {
  acquired?: boolean;
  reason?: string | null;
  lock?: EditLockView | null;
};

export function getTripEditLock(tripId: string): Promise<EditLockView> {
  return apiFetch<EditLockView>(`/trips/${tripId}/edit-lock`);
}

export async function acquireTripEditLock(
  tripId: string
): Promise<AcquireEditLockResponse> {
  try {
    return await apiFetch<AcquireEditLockResponse>(`/trips/${tripId}/edit-lock`, {
      method: "POST",
      body: JSON.stringify({ scope: "itinerary" })
    });
  } catch (error) {
    if (
      error instanceof ApiError &&
      error.status === 409 &&
      (error.code === "edit_lock_conflict" || error.code === "locked_by_other_user")
    ) {
      const payload = error.payload as EditLockConflictPayload | null | undefined;
      return {
        acquired: false,
        reason: payload?.reason ?? "locked_by_other_user",
        lock: payload?.lock ?? null
      };
    }
    throw error;
  }
}

export function releaseTripEditLock(tripId: string): Promise<ReleaseEditLockResponse> {
  return apiFetch<ReleaseEditLockResponse>(`/trips/${tripId}/edit-lock`, {
    method: "DELETE",
    body: JSON.stringify({ scope: "itinerary" })
  });
}
