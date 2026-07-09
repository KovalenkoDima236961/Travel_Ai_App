"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslations } from "next-intl";
import { useAuth } from "@/components/auth/AuthProvider";
import { getErrorMessage } from "@/lib/utils";
import { loginSchema, safeNextPath, type LoginValues } from "../model/authModel";
import { AuthErrorBanner, AuthField, AuthSubmitButton } from "./formControls";

export function LoginForm() {
  const translate = useTranslations("auth");
  const router = useRouter();
  const searchParams = useSearchParams();
  const { login } = useAuth();
  const [apiError, setApiError] = useState<string | null>(null);

  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register
  } = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: "", password: "" }
  });

  async function onSubmit(values: LoginValues) {
    setApiError(null);
    try {
      await login({ email: values.email.trim().toLowerCase(), password: values.password });
      router.push(safeNextPath(searchParams.get("next")) ?? "/trips");
    } catch (error) {
      setApiError(getErrorMessage(error, "Could not log in."));
    }
  }

  return (
    <form className="mt-7 flex flex-col gap-4" onSubmit={handleSubmit(onSubmit)} noValidate>
      <AuthField
        label={translate("email")}
        id="email"
        type="email"
        placeholder="you@example.com"
        autoComplete="email"
        error={errors.email?.message}
        {...register("email")}
      />
      <AuthField
        label={translate("password")}
        id="password"
        type="password"
        placeholder="••••••••"
        autoComplete="current-password"
        error={errors.password?.message}
        {...register("password")}
      />
      {apiError ? <AuthErrorBanner message={apiError} /> : null}
      <AuthSubmitButton pending={isSubmitting}>
        {isSubmitting ? translate("loggingIn") : translate("login")}
      </AuthSubmitButton>
    </form>
  );
}
