"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useAuth } from "@/components/auth/AuthProvider";
import { PageContainer } from "@/components/layout/PageContainer";
import { Button, buttonStyles } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { getErrorMessage } from "@/lib/utils";

const registerSchema = z
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

type RegisterValues = z.infer<typeof registerSchema>;

export default function RegisterPage() {
  const router = useRouter();
  const { register: registerAccount } = useAuth();
  const [apiError, setApiError] = useState<string | null>(null);

  const form = useForm<RegisterValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      email: "",
      password: "",
      confirmPassword: ""
    }
  });

  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register
  } = form;

  async function onSubmit(values: RegisterValues) {
    setApiError(null);
    try {
      await registerAccount({
        email: values.email.trim().toLowerCase(),
        password: values.password
      });
      router.push("/trips");
    } catch (error) {
      setApiError(getErrorMessage(error, "Could not create account."));
    }
  }

  return (
    <PageContainer className="max-w-lg">
      <div className="mb-8">
        <p className="text-sm font-semibold uppercase text-primary-700">Account</p>
        <h1 className="mt-2 text-3xl font-semibold text-slate-950">Register</h1>
      </div>

      <Card>
        <form className="space-y-5" onSubmit={handleSubmit(onSubmit)}>
          <label className="block">
            <span className="text-sm font-medium text-slate-800">Email</span>
            <span className="mt-2 block">
              <Input
                autoComplete="email"
                id="email"
                type="email"
                aria-invalid={Boolean(errors.email)}
                {...register("email")}
              />
            </span>
            {errors.email?.message ? (
              <span className="mt-2 block text-sm text-red-700">{errors.email.message}</span>
            ) : null}
          </label>

          <label className="block">
            <span className="text-sm font-medium text-slate-800">Password</span>
            <span className="mt-2 block">
              <Input
                autoComplete="new-password"
                id="password"
                type="password"
                aria-invalid={Boolean(errors.password)}
                {...register("password")}
              />
            </span>
            {errors.password?.message ? (
              <span className="mt-2 block text-sm text-red-700">{errors.password.message}</span>
            ) : null}
          </label>

          <label className="block">
            <span className="text-sm font-medium text-slate-800">Confirm password</span>
            <span className="mt-2 block">
              <Input
                autoComplete="new-password"
                id="confirmPassword"
                type="password"
                aria-invalid={Boolean(errors.confirmPassword)}
                {...register("confirmPassword")}
              />
            </span>
            {errors.confirmPassword?.message ? (
              <span className="mt-2 block text-sm text-red-700">
                {errors.confirmPassword.message}
              </span>
            ) : null}
          </label>

          {apiError ? (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800" role="alert">
              {apiError}
            </div>
          ) : null}

          <Button className="w-full" disabled={isSubmitting} type="submit">
            {isSubmitting ? "Creating account..." : "Register"}
          </Button>
        </form>
      </Card>

      <p className="mt-5 text-sm text-slate-600">
        Already registered?{" "}
        <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/login">
          Login
        </Link>
      </p>
    </PageContainer>
  );
}
