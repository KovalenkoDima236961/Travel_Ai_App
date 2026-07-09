import { z } from "zod";

export const loginSchema = z.object({
  email: z.string().trim().email("Enter a valid email address"),
  password: z.string().min(1, "Password is required")
});

export type LoginValues = z.infer<typeof loginSchema>;

export const registerSchema = z
  .object({
    email: z.string().trim().email("Enter a valid email address"),
    password: z
      .string()
      .min(8, "Password must be at least 8 characters")
      .regex(/[a-z]/, "Password must include a lowercase letter")
      .regex(/[A-Z]/, "Password must include an uppercase letter")
      .regex(/[0-9]/, "Password must include a digit"),
    confirmPassword: z.string().min(1, "Confirm your password")
  })
  .refine((values) => values.password === values.confirmPassword, {
    path: ["confirmPassword"],
    message: "Passwords must match"
  });

export type RegisterValues = z.infer<typeof registerSchema>;

/** Only allow in-app redirects so `?next=` can't bounce to an external origin. */
export function safeNextPath(value: string | null) {
  if (!value || !value.startsWith("/") || value.startsWith("//")) {
    return null;
  }

  return value;
}
