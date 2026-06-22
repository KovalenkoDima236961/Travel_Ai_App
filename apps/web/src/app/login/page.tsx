"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { zodResolver } from "@hookform/resolvers/zod";
import { Suspense, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useAuth } from "@/components/auth/AuthProvider";
import { PageContainer } from "@/components/layout/PageContainer";
import { Button, buttonStyles } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { getErrorMessage } from "@/lib/utils";

const loginSchema = z.object({
  email: z.string().trim().email("Enter a valid email address"),
  password: z.string().min(1, "Password is required")
});

type LoginValues = z.infer<typeof loginSchema>;

export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <PageContainer className="max-w-lg">
          <div className="text-sm text-slate-600">Loading login...</div>
        </PageContainer>
      }
    >
      <LoginPageContent />
    </Suspense>
  );
}

function LoginPageContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { login } = useAuth();
  const [apiError, setApiError] = useState<string | null>(null);

  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: "",
      password: ""
    }
  });

  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    register
  } = form;

  async function onSubmit(values: LoginValues) {
    setApiError(null);
    try {
      await login({
        email: values.email.trim().toLowerCase(),
        password: values.password
      });
      router.push(safeNextPath(searchParams.get("next")) ?? "/trips");
    } catch (error) {
      setApiError(getErrorMessage(error, "Could not log in."));
    }
  }

  return (
    <PageContainer className="max-w-lg">
      <div className="mb-8">
        <p className="text-sm font-semibold uppercase text-primary-700">Account</p>
        <h1 className="mt-2 text-3xl font-semibold text-slate-950">Login</h1>
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
                autoComplete="current-password"
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

          {apiError ? (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800" role="alert">
              {apiError}
            </div>
          ) : null}

          <Button className="w-full" disabled={isSubmitting} type="submit">
            {isSubmitting ? "Logging in..." : "Login"}
          </Button>
        </form>
      </Card>

      <p className="mt-5 text-sm text-slate-600">
        No account yet?{" "}
        <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/register">
          Register
        </Link>
      </p>
    </PageContainer>
  );
}

function safeNextPath(value: string | null) {
  if (!value || !value.startsWith("/") || value.startsWith("//")) {
    return null;
  }

  return value;
}
