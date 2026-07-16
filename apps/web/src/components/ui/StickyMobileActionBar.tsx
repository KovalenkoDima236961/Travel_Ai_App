"use client";

import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import { cn } from "@/shared/lib/cn";
import { ButtonSpinner } from "./ButtonSpinner";

type StickyMobileActionBarProps = {
  primaryLabel: string;
  onPrimary: () => void;
  cancelLabel?: string;
  onCancel?: () => void;
  primaryDisabled?: boolean;
  pending?: boolean;
  pendingLabel?: string;
  className?: string;
  mobileOnly?: boolean;
};

export function StickyMobileActionBar({
  primaryLabel,
  onPrimary,
  cancelLabel,
  onCancel,
  primaryDisabled = false,
  pending = false,
  pendingLabel,
  className,
  mobileOnly = true
}: StickyMobileActionBarProps) {
  const common = useTranslations("common");
  const accessibility = useTranslations("accessibility");
  return (
    <div
      aria-label={accessibility("mobileActions")}
      className={cn(
        "sticky bottom-0 z-30 -mx-4 mt-5 border-t border-slate-200 bg-white/95 px-4 py-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] shadow-[0_-10px_25px_rgba(15,23,42,0.08)] backdrop-blur",
        mobileOnly && "md:hidden",
        className
      )}
      role="group"
    >
      <div className="mx-auto flex max-w-xl gap-2">
        {onCancel ? (
          <Button className="flex-1" disabled={pending} onClick={onCancel} type="button" variant="secondary">
            {cancelLabel ?? common("cancel")}
          </Button>
        ) : null}
        <Button
          className="flex-1 gap-2"
          disabled={primaryDisabled || pending}
          onClick={onPrimary}
          type="button"
        >
          {pending ? <ButtonSpinner /> : null}
          {pending ? pendingLabel ?? common("saving") : primaryLabel}
        </Button>
      </div>
    </div>
  );
}
