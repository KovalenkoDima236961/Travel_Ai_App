import { apiFetch } from "@/shared/api/client";
import { queryKeys } from "@/lib/query-keys";
import type {
  RealWorldReadiness,
  RunVerificationActionInput,
  VerificationActionResult,
  VerificationScope
} from "@/types/verification";

export const verificationKeys = {
  all: ["verification"] as const,
  detail: (tripId: string) => queryKeys.trip.verification(tripId),
  section: (tripId: string, scope: VerificationScope) =>
    [...queryKeys.trip.verification(tripId), scope] as const
};

export function getTripVerification(tripId: string) {
  return apiFetch<RealWorldReadiness>(`/trips/${tripId}/verification`);
}

export function runVerificationAction(tripId: string, input: RunVerificationActionInput) {
  return apiFetch<VerificationActionResult>(`/trips/${tripId}/verification/actions`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}
