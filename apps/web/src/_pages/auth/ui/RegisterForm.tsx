"use client";

import { useRouter } from "next/navigation";
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useAuth } from "@/components/auth/AuthProvider";
import { getErrorMessage } from "@/lib/utils";
import { registerSchema, type RegisterValues } from "../model/authModel";
import { AuthErrorBanner, AuthField, AuthSubmitButton } from "./formControls";

export function RegisterForm() {
  const router = useRouter();
  const { register: registerAccount } = useAuth();
  const [apiError, setApiError] = useState<string | null>(null);

  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register
  } = useForm<RegisterValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: { email: "", password: "", confirmPassword: "" }
  });

  async function onSubmit(values: RegisterValues) {
    setApiError(null);
    try {
      await registerAccount({ email: values.email.trim().toLowerCase(), password: values.password });
      router.push("/trips");
    } catch (error) {
      setApiError(getErrorMessage(error, "Could not create account."));
    }
  }

  return (
    <form className="mt-7 flex flex-col gap-4" onSubmit={handleSubmit(onSubmit)} noValidate>
      <AuthField
        label="Email"
        id="email"
        type="email"
        placeholder="you@example.com"
        autoComplete="email"
        error={errors.email?.message}
        {...register("email")}
      />
      <AuthField
        label="Password"
        id="password"
        type="password"
        placeholder="••••••••"
        autoComplete="new-password"
        error={errors.password?.message}
        {...register("password")}
      />
      <AuthField
        label="Confirm password"
        id="confirmPassword"
        type="password"
        placeholder="••••••••"
        autoComplete="new-password"
        error={errors.confirmPassword?.message}
        {...register("confirmPassword")}
      />
      {apiError ? <AuthErrorBanner message={apiError} /> : null}
      <AuthSubmitButton pending={isSubmitting}>
        {isSubmitting ? "Creating account…" : "Create account"}
      </AuthSubmitButton>
    </form>
  );
}
