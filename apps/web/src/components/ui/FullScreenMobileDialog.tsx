"use client";

import { useEffect, useId, useRef, type ReactNode } from "react";
import { Button } from "@/shared/ui/button";
import { cn } from "@/shared/lib/cn";

type FullScreenMobileDialogProps = {
  title: string;
  children: ReactNode;
  onClose: () => void;
  closeLabel: string;
  description?: string;
  className?: string;
};

/**
 * A small dialog shell for long workflows: full-screen on phones and a
 * constrained, scrollable dialog on larger screens. It preserves the opener's
 * focus and gives the close action a stable, reachable position.
 */
export function FullScreenMobileDialog({
  title,
  children,
  onClose,
  closeLabel,
  description,
  className
}: FullScreenMobileDialogProps) {
  const titleId = useId();
  const descriptionId = useId();
  const closeRef = useRef<HTMLButtonElement>(null);
  const dialogRef = useRef<HTMLElement>(null);

  useEffect(() => {
    const previouslyFocused = document.activeElement as HTMLElement | null;
    closeRef.current?.focus();

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        event.preventDefault();
        onClose();
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

    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      previouslyFocused?.focus();
    };
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-[70] bg-slate-950/45 sm:flex sm:items-center sm:justify-center sm:p-5">
      <section
        aria-describedby={description ? descriptionId : undefined}
        aria-labelledby={titleId}
        aria-modal="true"
        className={cn(
          "flex h-[100dvh] w-full flex-col bg-white shadow-2xl sm:h-auto sm:max-h-[92dvh] sm:max-w-2xl sm:rounded-2xl",
          className
        )}
        ref={dialogRef}
        role="dialog"
      >
        <header className="flex shrink-0 items-start justify-between gap-4 border-b border-slate-200 bg-white px-4 py-3 sm:rounded-t-2xl sm:px-6 sm:py-4">
          <div className="min-w-0">
            <h2 className="text-lg font-semibold text-slate-950" id={titleId}>
              {title}
            </h2>
            {description ? (
              <p className="mt-1 text-sm leading-6 text-slate-600" id={descriptionId}>
                {description}
              </p>
            ) : null}
          </div>
          <Button
            aria-label={closeLabel}
            className="shrink-0"
            onClick={onClose}
            ref={closeRef}
            type="button"
            variant="ghost"
          >
            {closeLabel}
          </Button>
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-4 pb-[calc(env(safe-area-inset-bottom)+1.25rem)] sm:p-6">
          {children}
        </div>
      </section>
    </div>
  );
}
