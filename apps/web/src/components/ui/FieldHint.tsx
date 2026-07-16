import type { ReactNode } from "react";
import { cn } from "@/shared/lib/cn";

export function FieldHint({
  id,
  children,
  className
}: {
  id: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <p className={cn("text-xs leading-5 text-slate-500", className)} id={id}>
      {children}
    </p>
  );
}
