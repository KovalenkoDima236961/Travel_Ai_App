import { TextareaHTMLAttributes, forwardRef } from "react";
import { cn } from "@/shared/lib/cn";

type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement>;

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  { className, ...props },
  ref
) {
  return (
    <textarea
      ref={ref}
      className={cn(
        "min-h-28 w-full rounded-md border border-slate-300 bg-white px-3 py-3 text-sm text-slate-950 outline-none transition placeholder:text-slate-400 focus:border-primary-600 focus:ring-2 focus:ring-primary-100 disabled:cursor-not-allowed disabled:bg-slate-100",
        className
      )}
      {...props}
    />
  );
});
