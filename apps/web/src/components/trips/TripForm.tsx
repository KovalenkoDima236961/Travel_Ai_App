"use client";

import { useRouter } from "next/navigation";
import type { ReactNode } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { createTrip, tripKeys } from "@/lib/api/trips";
import { getErrorMessage } from "@/lib/utils";

const interestOptions = [
  { value: "food", label: "Food" },
  { value: "history", label: "History" },
  { value: "hidden_gems", label: "Hidden gems" },
  { value: "nature", label: "Nature" },
  { value: "nightlife", label: "Nightlife" },
  { value: "culture", label: "Culture" },
  { value: "budget", label: "Budget" },
  { value: "shopping", label: "Shopping" }
];

const paceOptions = [
  { value: "relaxed", label: "Relaxed" },
  { value: "balanced", label: "Balanced" },
  { value: "packed", label: "Intensive" }
] as const;

const tripFormSchema = z.object({
  destination: z.string().trim().min(1, "Destination is required"),
  startDate: z.string().optional(),
  days: z.number().int().min(1, "Days must be at least 1").max(30, "Days must be 30 or fewer"),
  budgetAmount: z.number().min(0, "Budget must be zero or greater").optional(),
  budgetCurrency: z
    .string()
    .trim()
    .length(3, "Use a 3-letter currency code"),
  travelers: z.number().int().min(1, "Travelers must be at least 1"),
  interests: z.array(z.string()),
  pace: z.enum(["relaxed", "balanced", "packed"])
});

type TripFormValues = z.infer<typeof tripFormSchema>;

export function TripForm() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const form = useForm<TripFormValues>({
    resolver: zodResolver(tripFormSchema),
    defaultValues: {
      destination: "",
      startDate: "",
      days: 4,
      budgetCurrency: "EUR",
      travelers: 2,
      interests: [],
      pace: "balanced"
    }
  });

  const createMutation = useMutation({
    mutationFn: createTrip,
    onSuccess: async (trip) => {
      await queryClient.invalidateQueries({ queryKey: tripKeys.lists() });
      router.push(`/trips/${trip.id}`);
    }
  });

  const {
    register,
    handleSubmit,
    formState: { errors }
  } = form;

  function onSubmit(values: TripFormValues) {
    createMutation.mutate({
      destination: values.destination,
      startDate: values.startDate || undefined,
      days: values.days,
      budgetAmount: values.budgetAmount,
      budgetCurrency: values.budgetCurrency.toUpperCase(),
      travelers: values.travelers,
      interests: values.interests ?? [],
      pace: values.pace
    });
  }

  return (
    <Card>
      <form className="space-y-6" onSubmit={handleSubmit(onSubmit)}>
        <div className="grid gap-5 md:grid-cols-2">
          <Field label="Destination" error={errors.destination?.message}>
            <Input
              id="destination"
              placeholder="Rome"
              aria-invalid={Boolean(errors.destination)}
              {...register("destination")}
            />
          </Field>

          <Field label="Start date" error={errors.startDate?.message}>
            <Input id="startDate" type="date" {...register("startDate")} />
          </Field>

          <Field label="Days" error={errors.days?.message}>
            <Input
              id="days"
              type="number"
              min={1}
              max={30}
              aria-invalid={Boolean(errors.days)}
              {...register("days", { valueAsNumber: true })}
            />
          </Field>

          <Field label="Travelers" error={errors.travelers?.message}>
            <Input
              id="travelers"
              type="number"
              min={1}
              aria-invalid={Boolean(errors.travelers)}
              {...register("travelers", { valueAsNumber: true })}
            />
          </Field>

          <Field label="Budget amount" error={errors.budgetAmount?.message}>
            <Input
              id="budgetAmount"
              type="number"
              min={0}
              step="0.01"
              placeholder="600"
              aria-invalid={Boolean(errors.budgetAmount)}
              {...register("budgetAmount", {
                setValueAs: (value) => (value === "" ? undefined : Number(value))
              })}
            />
          </Field>

          <Field label="Budget currency" error={errors.budgetCurrency?.message}>
            <Select id="budgetCurrency" aria-invalid={Boolean(errors.budgetCurrency)} {...register("budgetCurrency")}>
              <option value="EUR">EUR</option>
              <option value="USD">USD</option>
              <option value="GBP">GBP</option>
              <option value="CZK">CZK</option>
            </Select>
          </Field>

          <Field label="Pace" error={errors.pace?.message}>
            <Select id="pace" aria-invalid={Boolean(errors.pace)} {...register("pace")}>
              {paceOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </Select>
          </Field>
        </div>

        <fieldset>
          <legend className="text-sm font-medium text-slate-800">Interests</legend>
          <div className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            {interestOptions.map((option) => (
              <label
                key={option.value}
                className="flex min-h-11 items-center gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm font-medium text-slate-700"
              >
                <input
                  className="h-4 w-4 rounded border-slate-300 text-primary-600 focus:ring-primary-600"
                  type="checkbox"
                  value={option.value}
                  {...register("interests")}
                />
                {option.label}
              </label>
            ))}
          </div>
          {errors.interests?.message ? (
            <p className="mt-2 text-sm text-red-700">{errors.interests.message}</p>
          ) : null}
        </fieldset>

        {createMutation.isError ? (
          <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800" role="alert">
            {getErrorMessage(createMutation.error, "Could not create trip.")}
          </div>
        ) : null}

        <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <Button
            disabled={createMutation.isPending}
            type="button"
            variant="secondary"
            onClick={() => router.push("/trips")}
          >
            Cancel
          </Button>
          <Button disabled={createMutation.isPending} type="submit">
            {createMutation.isPending ? "Creating..." : "Create trip"}
          </Button>
        </div>
      </form>
    </Card>
  );
}

type FieldProps = {
  label: string;
  error?: string;
  children: ReactNode;
};

function Field({ label, error, children }: FieldProps) {
  return (
    <label className="block">
      <span className="text-sm font-medium text-slate-800">{label}</span>
      <span className="mt-2 block">{children}</span>
      {error ? <span className="mt-2 block text-sm text-red-700">{error}</span> : null}
    </label>
  );
}
