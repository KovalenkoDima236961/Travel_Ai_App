"use client";

import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import { ButtonSpinner } from "./ButtonSpinner";

type RetryButtonProps = {
  onRetry: () => void;
  label?: string;
  pending?: boolean;
  disabled?: boolean;
};

export function RetryButton({ onRetry, label, pending = false, disabled = false }: RetryButtonProps) {
  const t = useTranslations("common");
  return (
    <Button disabled={disabled || pending} onClick={onRetry} size="sm" type="button" variant="secondary">
      {pending ? <ButtonSpinner className="mr-2" /> : null}
      {pending ? t("retrying") : label ?? t("retry")}
    </Button>
  );
}
