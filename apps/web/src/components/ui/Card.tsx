import { HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

type CardProps = HTMLAttributes<HTMLDivElement>;

export function Card({ className, ...props }: CardProps) {
  return (
    <div
      className={cn("rounded-lg border border-slate-200 bg-white p-5 shadow-soft", className)}
      {...props}
    />
  );
}
