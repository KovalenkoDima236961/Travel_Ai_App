import type { useTranslations } from "next-intl";

const ERROR_KEYS = {
  unauthorized: "unauthorized",
  forbidden: "forbidden",
  not_found: "notFound",
  validation_error: "validationError",
  itinerary_conflict: "itineraryConflict",
  edit_lock_conflict: "editLockConflict",
  workspace_policy_blocking_violation: "workspacePolicyBlockingViolation",
  provider_rate_limited: "providerRateLimited",
  provider_quota_exceeded: "providerQuotaExceeded",
  ai_generation_failed: "aiGenerationFailed",
  repair_proposal_conflict: "repairProposalConflict"
} as const;

type Translator = ReturnType<typeof useTranslations<"errors">>;

export function getLocalizedErrorMessage(
  code: string | null | undefined,
  translate: Translator,
  backendMessage?: string
): string {
  const key = code ? ERROR_KEYS[code as keyof typeof ERROR_KEYS] : undefined;
  if (key) {
    return translate(key);
  }
  return backendMessage?.trim() || translate("generic");
}
