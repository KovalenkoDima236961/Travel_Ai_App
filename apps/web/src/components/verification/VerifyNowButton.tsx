"use client";

import { useState } from "react";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { Button, buttonStyles } from "@/shared/ui/button";
import { getErrorMessage } from "@/lib/utils";
import { useRunVerificationAction } from "@/hooks/useTripVerification";
import type { VerificationAction, VerificationScope } from "@/types/verification";

type VerifyNowButtonProps = {
  tripId: string;
  action: VerificationAction;
  scope: VerificationScope;
  entityType?: string;
  entityId?: string;
  onComplete?: (message: string) => void;
};

export function VerifyNowButton({ tripId, action, scope, entityType, entityId, onComplete }: VerifyNowButtonProps) {
  const mutation = useRunVerificationAction(tripId);
  const [error, setError] = useState<string | null>(null);
  const t = useTranslations("verification");
  const isNavigationAction = [
    "review_opening_hours",
    "check_availability",
    "add_accommodation",
    "attach_place",
    "open_route",
    "open_budget",
    "open_itinerary_item"
  ].includes(action.type);

  async function run() {
    setError(null);
    try {
      const result = await mutation.mutateAsync({
        actionType: action.type,
        scope,
        entityType,
        entityId
      });
      onComplete?.(result.message);
      if (result.status === "failed") {
        setError(result.message);
      }
    } catch (cause) {
      setError(getErrorMessage(cause, t("refreshFailed")));
    }
  }

  if (isNavigationAction) {
    return (
      <Link className={buttonStyles({ size: "sm", variant: "secondary" })} href={action.href}>
        {t(`actions.${action.type}`)}
      </Link>
    );
  }

  return (
    <span className="inline-flex flex-col items-start gap-1">
      <Button disabled={mutation.isPending} onClick={() => void run()} size="sm" type="button" variant="secondary">
        {mutation.isPending ? t("checking") : t(`actions.${action.type}`)}
      </Button>
      {error ? <span className="text-xs text-red-700">{error}</span> : null}
    </span>
  );
}
