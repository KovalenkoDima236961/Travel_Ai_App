"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useTranslations } from "next-intl";
import { createTrip, tripKeys } from "@/lib/api/trips";
import { getErrorMessage } from "@/lib/utils";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import {
  createDefaultTripRoute,
  getRouteValidationWarnings,
  TripRouteBuilder
} from "@/components/routes/TripRouteBuilder";
import { CheckCircleIcon, MapPinIcon, SparklesIcon } from "./icons";

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

const tripFormSchema = z
  .object({
    tripMode: z.enum(["single_destination", "multi_destination"]),
    destination: z.string().trim(),
    workspaceId: z.string().optional(),
    startDate: z.string().optional(),
    days: z.number().int().min(1, "Days must be at least 1").max(30, "Days must be 30 or fewer"),
    budgetAmount: z.number().min(0, "Budget must be zero or greater").optional(),
    budgetCurrency: z.string().trim().length(3, "Use a 3-letter currency code"),
    travelers: z.number().int().min(1, "Travelers must be at least 1"),
    interests: z.array(z.string()),
    pace: z.enum(["relaxed", "balanced", "packed"])
  })
  .superRefine((values, context) => {
    if (values.tripMode === "single_destination" && values.destination.trim().length === 0) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["destination"],
        message: "Destination is required"
      });
    }
  });

type TripFormValues = z.infer<typeof tripFormSchema>;

// Warm field primitives — mirror the settings/trips slices, scoped to this screen.
const FIELD_LABEL = "block text-[13.5px] font-semibold text-cocoa-700";
const FIELD_INPUT =
  "mt-2 h-12 w-full rounded-xl border border-sand-400 bg-[#FFFDFA] px-3.5 text-[14.5px] text-cocoa-900 outline-none transition placeholder:text-cocoa-400 focus:border-clay focus:ring-[3px] focus:ring-clay-tint";
const FIELD_ERROR = "mt-1.5 block text-[13px] text-clay-deep";

export function CreateTripForm() {
  const translate = useTranslations("trips");
  const translateCommon = useTranslations("common");
  const router = useRouter();
  const queryClient = useQueryClient();
  const { currentScope, currentWorkspace, editableWorkspaces } = useWorkspaces();
  const [route, setRoute] = useState(createDefaultTripRoute);
  const [routeError, setRouteError] = useState<string | null>(null);

  const form = useForm<TripFormValues>({
    resolver: zodResolver(tripFormSchema),
    defaultValues: {
      tripMode: "single_destination",
      destination: "",
      workspaceId: "",
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
    setValue,
    watch,
    formState: { errors }
  } = form;

  // `interests` is driven by the pill buttons via setValue (no input element), so
  // register it explicitly to guarantee it's tracked and always present in the
  // submitted payload — independent of defaultValues init semantics.
  useEffect(() => {
    register("interests");
  }, [register]);

  useEffect(() => {
    if (
      currentScope === "workspace" &&
      currentWorkspace &&
      editableWorkspaces.some((workspace) => workspace.id === currentWorkspace.id)
    ) {
      setValue("workspaceId", currentWorkspace.id);
    }
  }, [currentScope, currentWorkspace, editableWorkspaces, setValue]);

  const selectedInterests = watch("interests") ?? [];
  const selectedTripMode = watch("tripMode");
  const selectedDays = watch("days") ?? 1;
  const selectedCurrency = watch("budgetCurrency") ?? "EUR";

  function toggleInterest(value: string) {
    const next = selectedInterests.includes(value)
      ? selectedInterests.filter((item) => item !== value)
      : [...selectedInterests, value];
    setValue("interests", next, { shouldDirty: true });
  }

  function onSubmit(values: TripFormValues) {
    if (values.tripMode === "multi_destination") {
      const blockingWarning = getRouteValidationWarnings(route, values.days).find(
        (warning) => warning.severity === "error"
      );
      if (blockingWarning) {
        setRouteError(blockingWarning.message);
        return;
      }
      setRouteError(null);
    }

    const routeDestination = deriveRouteDestination(route);
    createMutation.mutate({
      destination:
        values.tripMode === "multi_destination"
          ? values.destination.trim() || routeDestination
          : values.destination,
      tripType: values.tripMode,
      route: values.tripMode === "multi_destination" ? route : null,
      workspaceId: values.workspaceId || undefined,
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
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex flex-col gap-9 rounded-[22px] border border-sand-300 bg-white px-6 py-9 shadow-[0_1px_2px_rgba(34,26,20,0.04),0_14px_36px_rgba(34,26,20,0.06)] sm:px-10"
    >
      <section>
        <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">
          {translate("basics")}
        </h2>
        <div className="mt-5 flex flex-col gap-[18px]">
          <div className="inline-flex w-fit gap-1 rounded-full border border-sand-300 bg-sand-50 p-1">
            {[
              { value: "single_destination", label: "Single destination" },
              { value: "multi_destination", label: "Multi-destination route" }
            ].map((option) => (
              <button
                key={option.value}
                type="button"
                aria-pressed={selectedTripMode === option.value}
                onClick={() =>
                  setValue("tripMode", option.value as TripFormValues["tripMode"], {
                    shouldDirty: true
                  })
                }
                className={
                  selectedTripMode === option.value
                    ? "h-10 rounded-full bg-cocoa-900 px-5 text-[14px] font-semibold text-sand-150"
                    : "h-10 rounded-full px-5 text-[14px] font-medium text-cocoa-500 transition hover:bg-sand-200 hover:text-cocoa-900"
                }
              >
                {option.label}
              </button>
            ))}
          </div>

          <label className="block">
            <span className={FIELD_LABEL}>
              {selectedTripMode === "multi_destination" ? "Trip name or region" : translate("destination")}
            </span>
            <span className="mt-2 flex h-[52px] items-center gap-2.5 rounded-[14px] border border-sand-400 bg-[#FFFDFA] px-4 transition focus-within:border-clay focus-within:ring-[3px] focus-within:ring-clay-tint">
              <MapPinIcon className="h-[19px] w-[19px] shrink-0 text-clay" />
              <input
                id="destination"
                placeholder="City, region, or country"
                aria-invalid={Boolean(errors.destination)}
                {...register("destination")}
                className="flex-1 border-none bg-transparent text-[16px] text-cocoa-900 outline-none placeholder:text-cocoa-400"
              />
            </span>
            {errors.destination?.message ? (
              <span className={FIELD_ERROR}>{errors.destination.message}</span>
            ) : null}
          </label>

          <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-3">
            <label className="block">
              <span className={FIELD_LABEL}>{translate("startDate")}</span>
              <input id="startDate" type="date" className={FIELD_INPUT} {...register("startDate")} />
              {errors.startDate?.message ? (
                <span className={FIELD_ERROR}>{errors.startDate.message}</span>
              ) : null}
            </label>

            <label className="block">
              <span className={FIELD_LABEL}>{translate("days")}</span>
              <input
                id="days"
                type="number"
                min={1}
                max={30}
                aria-invalid={Boolean(errors.days)}
                className={FIELD_INPUT}
                {...register("days", { valueAsNumber: true })}
              />
              {errors.days?.message ? <span className={FIELD_ERROR}>{errors.days.message}</span> : null}
            </label>

            <label className="block">
              <span className={FIELD_LABEL}>{translate("tripScope")}</span>
              <select
                id="workspaceId"
                aria-invalid={Boolean(errors.workspaceId)}
                className={FIELD_INPUT}
                {...register("workspaceId")}
              >
                <option value="">{translate("personalTrip")}</option>
                {editableWorkspaces.map((workspace) => (
                  <option key={workspace.id} value={workspace.id}>
                    {workspace.name}
                  </option>
                ))}
              </select>
              {errors.workspaceId?.message ? (
                <span className={FIELD_ERROR}>{errors.workspaceId.message}</span>
              ) : null}
            </label>
          </div>
        </div>
      </section>

      {selectedTripMode === "multi_destination" ? (
        <section className="border-t border-sand-200 pt-8">
          <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">
            Multi-destination route
          </h2>
          <div className="mt-5">
            <TripRouteBuilder
              value={route}
              onChange={setRoute}
              totalDays={selectedDays}
              currency={selectedCurrency}
            />
          </div>
          {routeError ? <p className={FIELD_ERROR}>{routeError}</p> : null}
        </section>
      ) : null}

      <section className="border-t border-sand-200 pt-8">
        <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">
          {translate("travelersBudget")}
        </h2>
        <div className="mt-5 grid grid-cols-1 gap-3.5 sm:grid-cols-3">
          <label className="block">
            <span className={FIELD_LABEL}>{translate("travelers")}</span>
            <input
              id="travelers"
              type="number"
              min={1}
              aria-invalid={Boolean(errors.travelers)}
              className={FIELD_INPUT}
              {...register("travelers", { valueAsNumber: true })}
            />
            {errors.travelers?.message ? (
              <span className={FIELD_ERROR}>{errors.travelers.message}</span>
            ) : null}
          </label>

          <label className="block">
            <span className={FIELD_LABEL}>
              {translate("budget")}{" "}
              <span className="font-normal text-[#A08D78]">
                ({translateCommon("optional")})
              </span>
            </span>
            <input
              id="budgetAmount"
              type="number"
              min={0}
              step="0.01"
              placeholder="600"
              aria-invalid={Boolean(errors.budgetAmount)}
              className={FIELD_INPUT}
              {...register("budgetAmount", {
                setValueAs: (value) => (value === "" ? undefined : Number(value))
              })}
            />
            {errors.budgetAmount?.message ? (
              <span className={FIELD_ERROR}>{errors.budgetAmount.message}</span>
            ) : null}
          </label>

          <label className="block">
            <span className={FIELD_LABEL}>{translate("currency")}</span>
            <select
              id="budgetCurrency"
              aria-invalid={Boolean(errors.budgetCurrency)}
              className={FIELD_INPUT}
              {...register("budgetCurrency")}
            >
              <option value="EUR">EUR</option>
              <option value="USD">USD</option>
              <option value="GBP">GBP</option>
              <option value="CZK">CZK</option>
            </select>
            {errors.budgetCurrency?.message ? (
              <span className={FIELD_ERROR}>{errors.budgetCurrency.message}</span>
            ) : null}
          </label>
        </div>
      </section>

      <section className="border-t border-sand-200 pt-8">
        <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">
          {translate("style")}
        </h2>

        <p className="mt-5 text-[13.5px] font-semibold text-cocoa-700">
          {translate("pace")}
        </p>
        <div className="mt-2 inline-flex gap-1 rounded-full border border-sand-300 bg-sand-50 p-1">
          {paceOptions.map((option) => (
            <label key={option.value} className="cursor-pointer">
              <input type="radio" value={option.value} className="peer sr-only" {...register("pace")} />
              <span className="flex h-[38px] items-center rounded-full px-5 text-[14px] font-medium text-cocoa-500 transition hover:bg-sand-200 peer-checked:bg-cocoa-900 peer-checked:font-semibold peer-checked:text-sand-150 peer-checked:hover:bg-cocoa-900">
                {option.label}
              </span>
            </label>
          ))}
        </div>

        <p className="mt-6 text-[13.5px] font-semibold text-cocoa-700">
          {translate("interests")}
        </p>
        <div className="mt-2.5 flex flex-wrap gap-2">
          {interestOptions.map((option) => {
            const selected = selectedInterests.includes(option.value);
            return (
              <button
                key={option.value}
                type="button"
                aria-pressed={selected}
                onClick={() => toggleInterest(option.value)}
                className={
                  selected
                    ? "inline-flex h-10 items-center gap-1.5 rounded-full border border-clay bg-clay-tint px-[18px] text-[14px] font-semibold text-clay-deep transition"
                    : "inline-flex h-10 items-center rounded-full border border-sand-400 bg-white px-[18px] text-[14px] font-medium text-cocoa-500 transition hover:border-sand-600 hover:text-cocoa-900"
                }
              >
                {selected ? <CheckCircleIcon className="h-[13px] w-[13px]" /> : null}
                {option.label}
              </button>
            );
          })}
        </div>
        {errors.interests?.message ? <p className={FIELD_ERROR}>{errors.interests.message}</p> : null}
      </section>

      {createMutation.isError ? (
        <div
          role="alert"
          className="rounded-2xl border border-clay/30 bg-clay-tint/50 px-4 py-3 text-[14px] text-clay-deep"
        >
          {getErrorMessage(createMutation.error, "Could not create trip.")}
        </div>
      ) : null}

      <div className="flex items-center justify-end gap-3 border-t border-sand-200 pt-7">
        <button
          type="button"
          disabled={createMutation.isPending}
          onClick={() => router.push("/trips")}
          className="inline-flex h-12 items-center rounded-full px-[22px] text-[15px] font-medium text-cocoa-500 transition hover:bg-sand-200 hover:text-cocoa-900 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {translateCommon("cancel")}
        </button>
        <button
          type="submit"
          disabled={createMutation.isPending}
          className="inline-flex h-12 items-center gap-2.5 rounded-full bg-clay px-[26px] text-[15px] font-semibold text-sand-100 shadow-[0_8px_20px_rgba(192,91,59,0.25)] transition hover:bg-clay-dark disabled:cursor-not-allowed disabled:opacity-60"
        >
          <SparklesIcon className="h-[17px] w-[17px]" />
          {createMutation.isPending ? translate("creating") : translate("create")}
        </button>
      </div>
    </form>
  );
}

function deriveRouteDestination(route: ReturnType<typeof createDefaultTripRoute>) {
  const firstStop = route.stops.find((stop) => stop.destination.trim().length > 0);
  if (!firstStop) {
    return "Multi-destination route";
  }
  const sameCountry =
    firstStop.country &&
    route.stops.every((stop) => (stop.country ?? "").trim() === firstStop.country?.trim());
  if (sameCountry) {
    return `${firstStop.country} route`;
  }
  return `${firstStop.destination} route`;
}
