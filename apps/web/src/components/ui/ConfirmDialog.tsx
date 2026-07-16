"use client";

import { useEffect, useId, useRef, type ReactNode } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import { cn } from "@/shared/lib/cn";
import { ButtonSpinner } from "./ButtonSpinner";

type ConfirmDialogProps = {
  open: boolean;
  title: string;
  description: ReactNode;
  confirmLabel: string;
  cancelLabel?: string;
  tone?: "default" | "danger";
  pending?: boolean;
  error?: string | null;
  children?: ReactNode;
  closeOnBackdrop?: boolean;
  onCancel: () => void;
  onConfirm: () => void;
};

export function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel,
  cancelLabel,
  tone = "default",
  pending = false,
  error,
  children,
  closeOnBackdrop = false,
  onCancel,
  onConfirm
}: ConfirmDialogProps) {
  const t = useTranslations("common");
  const titleId = useId();
  const descriptionId = useId();
  const cancelRef = useRef<HTMLButtonElement>(null);
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    const previouslyFocused = document.activeElement as HTMLElement | null;
    cancelRef.current?.focus();

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape" && !pending) {
        event.preventDefault();
        onCancel();
        return;
      }
      if (event.key !== "Tab" || !dialogRef.current) {
        return;
      }
      const focusable = Array.from(
        dialogRef.current.querySelectorAll<HTMLElement>(
          'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
        )
      );
      if (focusable.length === 0) {
        event.preventDefault();
        return;
      }
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      previouslyFocused?.focus();
    };
  }, [onCancel, open, pending]);

  if (!open) {
    return null;
  }

  return (
    <div
      className="fixed inset-0 z-[90] flex items-end justify-center bg-slate-950/45 p-0 sm:items-center sm:p-5"
      onMouseDown={(event) => {
        if (closeOnBackdrop && event.currentTarget === event.target && !pending) {
          onCancel();
        }
      }}
    >
      <div
        aria-describedby={descriptionId}
        aria-labelledby={titleId}
        aria-modal="true"
        className="max-h-[92vh] w-full overflow-y-auto rounded-t-2xl bg-white p-5 shadow-2xl sm:max-w-lg sm:rounded-2xl sm:p-6"
        ref={dialogRef}
        role="dialog"
      >
        <h2 className="text-lg font-semibold text-slate-950" id={titleId}>
          {title}
        </h2>
        <div className="mt-2 text-sm leading-6 text-slate-600" id={descriptionId}>
          {description}
        </div>
        {children ? <div className="mt-4">{children}</div> : null}
        {error ? (
          <p className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800" role="alert">
            {error}
          </p>
        ) : null}
        <div className="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Button disabled={pending} onClick={onCancel} ref={cancelRef} type="button" variant="secondary">
            {cancelLabel ?? t("cancel")}
          </Button>
          <Button
            className={cn("min-w-32", pending && "gap-2")}
            disabled={pending}
            onClick={onConfirm}
            type="button"
            variant={tone === "danger" ? "danger" : "primary"}
          >
            {pending ? <ButtonSpinner /> : null}
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}
