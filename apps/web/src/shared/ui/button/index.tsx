import { ButtonHTMLAttributes, forwardRef } from "react";
import { cn } from "@/shared/lib/cn";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
type ButtonSize = "sm" | "md";

type ButtonStyleOptions = {
  variant?: ButtonVariant;
  size?: ButtonSize;
  className?: string;
};

export function buttonStyles({
  variant = "primary",
  size = "md",
  className
}: ButtonStyleOptions = {}) {
  return cn(
    "inline-flex items-center justify-center rounded-md font-medium transition focus:outline-none focus:ring-2 focus:ring-primary-600 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60",
    size === "sm" ? "h-9 px-3 text-sm" : "h-11 px-4 text-sm",
    variant === "primary" && "bg-primary-600 text-white hover:bg-primary-700",
    variant === "secondary" &&
      "border border-slate-300 bg-white text-slate-800 hover:bg-slate-50",
    variant === "ghost" && "text-slate-700 hover:bg-slate-100",
    variant === "danger" && "bg-red-600 text-white hover:bg-red-700",
    className
  );
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & ButtonStyleOptions;

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant, size, className, type = "button", ...props },
  ref
) {
  return (
    <button
      className={buttonStyles({ variant, size, className })}
      ref={ref}
      type={type}
      {...props}
    />
  );
});
