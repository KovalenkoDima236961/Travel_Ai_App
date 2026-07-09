import { forwardRef, InputHTMLAttributes, ReactNode } from "react";
import { cn } from "@/shared/lib/cn";

type AuthFieldProps = InputHTMLAttributes<HTMLInputElement> & {
  label: string;
  error?: string;
};

/**
 * Editorial-styled text field for the auth screen. Designed to receive
 * react-hook-form's `register()` spread directly — the forwarded ref is the
 * field ref, and `error` drives both the message and the invalid border.
 */
export const AuthField = forwardRef<HTMLInputElement, AuthFieldProps>(function AuthField(
  { label, error, id, ...props },
  ref
) {
  return (
    <label className="block" htmlFor={id}>
      <span className="block text-[13.5px] font-semibold text-cocoa-700">{label}</span>
      <input
        ref={ref}
        id={id}
        aria-invalid={error ? true : undefined}
        className={cn(
          "mt-2 h-[50px] w-full rounded-[14px] border bg-[#FFFDFA] px-4 text-[15px] text-cocoa-900 outline-none transition placeholder:text-cocoa-400 focus:ring-[3px] focus:ring-clay-tint",
          error ? "border-[#C0553B] focus:border-[#C0553B]" : "border-sand-400 focus:border-clay"
        )}
        {...props}
      />
      {error ? <span className="mt-2 block text-[13px] text-[#B4442B]">{error}</span> : null}
    </label>
  );
});

export function AuthErrorBanner({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="rounded-[14px] border border-[#E7C4B8] bg-[#FBEDE7] px-4 py-3 text-[13.5px] text-clay-deep"
    >
      {message}
    </div>
  );
}

export function AuthSubmitButton({ pending, children }: { pending: boolean; children: ReactNode }) {
  return (
    <button
      type="submit"
      disabled={pending}
      className="mt-2 inline-flex h-[50px] w-full items-center justify-center rounded-full bg-clay text-[15px] font-semibold text-sand-100 shadow-[0_8px_20px_rgba(192,91,59,0.25)] transition hover:bg-clay-dark disabled:cursor-not-allowed disabled:opacity-70"
    >
      {children}
    </button>
  );
}
