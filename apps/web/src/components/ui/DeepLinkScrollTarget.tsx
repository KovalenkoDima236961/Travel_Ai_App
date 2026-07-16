"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import { useTranslations } from "next-intl";
import { cn } from "@/shared/lib/cn";

type DeepLinkScrollTargetProps = {
  itemId: string;
  targetId?: string | null;
  children: ReactNode;
  label?: string;
  className?: string;
  onMatch?: () => void;
};

export function DeepLinkScrollTarget({
  itemId,
  targetId,
  children,
  label,
  className,
  onMatch
}: DeepLinkScrollTargetProps) {
  const t = useTranslations("accessibility");
  const ref = useRef<HTMLDivElement>(null);
  const [highlighted, setHighlighted] = useState(false);
  const matches = Boolean(targetId) && targetId === itemId;

  useEffect(() => {
    if (!matches || !ref.current) {
      return;
    }
    const node = ref.current;
    node.scrollIntoView({ behavior: "smooth", block: "center" });
    node.focus({ preventScroll: true });
    setHighlighted(true);
    onMatch?.();
    const timer = window.setTimeout(() => setHighlighted(false), 2400);
    return () => window.clearTimeout(timer);
  }, [matches, onMatch]);

  return (
    <div
      aria-label={label}
      className={cn(
        "scroll-mt-24 rounded-lg outline-none transition-shadow duration-300",
        highlighted && "ring-2 ring-primary-600 ring-offset-2",
        className
      )}
      data-deep-link-highlighted={highlighted ? "true" : undefined}
      id={itemId}
      ref={ref}
      tabIndex={matches ? -1 : undefined}
    >
      {children}
      {matches ? <span className="sr-only" role="status">{t("deepLinkTargetOpened")}</span> : null}
    </div>
  );
}
