import { cn } from "@/shared/lib/cn";

export function InlineError({
  id,
  message,
  className
}: {
  id?: string;
  message: string;
  className?: string;
}) {
  return (
    <p className={cn("text-sm text-red-700", className)} id={id} role="alert">
      {message}
    </p>
  );
}
