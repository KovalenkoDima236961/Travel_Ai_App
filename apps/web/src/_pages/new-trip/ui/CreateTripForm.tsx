"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useTranslations } from "next-intl";
import { createGenerationJob } from "@/lib/api/generation-jobs";
import { createTrip, tripKeys } from "@/lib/api/trips";
import { getErrorMessage } from "@/lib/utils";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { useAuth } from "@/components/auth/AuthProvider";
import { useOnboardingState } from "@/hooks/useOnboardingState";
import { getMyPreferences, getMyProfile, userKeys } from "@/lib/api/user";
import {
  AdvancedTripPreferencesForm,
  type AdvancedTripPreferencesValue
} from "@/components/planning-constraints/AdvancedTripPreferencesForm";
import { PlanningConstraintsPreviewPanel } from "@/components/planning-constraints/PlanningConstraintsPreviewPanel";
import { RouteAlternativesPanel } from "@/components/route-alternatives";
import {
  createDefaultTripRoute,
  getRouteValidationWarnings,
  TripRouteBuilder
} from "@/components/routes/TripRouteBuilder";
import { usePlanningConstraintsPreview } from "@/hooks/usePlanningConstraintsPreview";
import { usePreferenceCompleteness } from "@/hooks/usePersonalization";
import { PreferenceReviewPrompt } from "@/components/personalization";
import type { TransportMode, TripRoute, TripStyle } from "@/entities/route/model";
import type { CreateTripInput } from "@/entities/trip/model";
import type { PlanningConstraintsPreviewRequest } from "@/types/planning-constraints";
import { CheckCircleIcon, MapPinIcon, SparklesIcon } from "./icons";
import { CreateTripReviewStep } from "./CreateTripReviewStep";
import { CreateTripStepper, type CreateTripStep } from "./CreateTripStepper";

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
    startDate: z.string().trim().min(1, "Choose a start date."),
    days: z
      .number()
      .int()
      .min(1, "Trip duration must be at least 1 day.")
      .max(30, "Trip duration must be 30 days or fewer."),
    budgetAmount: z.number().positive("Budget amount must be greater than 0.").optional(),
    budgetCurrency: z.string().trim().length(3, "Use a 3-letter currency code."),
    travelers: z.number().int().min(1, "Add at least one traveler."),
    interests: z.array(z.string()),
    pace: z.enum(["relaxed", "balanced", "packed"])
  })
  .superRefine((values, context) => {
    if (values.tripMode === "single_destination" && values.destination.length === 0) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["destination"],
        message: "Add at least one destination or route stop."
      });
    }
  });

type TripFormValues = z.infer<typeof tripFormSchema>;

const FIELD_LABEL = "block text-[13.5px] font-semibold text-cocoa-700";
const FIELD_INPUT =
  "mt-2 h-12 w-full rounded-xl border border-sand-400 bg-[#FFFDFA] px-3.5 text-[14.5px] text-cocoa-900 outline-none transition placeholder:text-cocoa-400 focus:border-clay focus:ring-[3px] focus:ring-clay-tint";
const FIELD_ERROR = "mt-1.5 block text-[13px] text-clay-deep";
export function CreateTripForm({
  initialTripMode = "single_destination"
}: {
  initialTripMode?: TripFormValues["tripMode"];
}) {
  const creationT = useTranslations("tripCreation");
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const onboarding = useOnboardingState(user?.id);
  const { currentScope, currentWorkspace, editableWorkspaces } = useWorkspaces();
  const preferenceDefaultsApplied = useRef(false);
  const profileQuery = useQuery({ queryKey: userKeys.profile(), queryFn: getMyProfile });
  const preferencesQuery = useQuery({ queryKey: userKeys.preferences(), queryFn: getMyPreferences });
  const completenessQuery = usePreferenceCompleteness();
  const planningPreviewMutation = usePlanningConstraintsPreview();
  const [currentStep, setCurrentStep] = useState<CreateTripStep>(1);
  const [route, setRoute] = useState(createDefaultTripRoute);
  const [routeError, setRouteError] = useState<string | null>(null);
  const [advancedPreferences, setAdvancedPreferences] = useState<AdvancedTripPreferencesValue>({
    budgetStrictness: "target",
    outputLanguage: "en",
    maxWalkingKmPerDay: 8,
    preferredModes: ["train", "public_transport"],
    avoidModes: [],
    carAvailable: false,
    maxTransferHoursPerDay: 4,
    tripStyles: []
  });

  const form = useForm<TripFormValues>({
    resolver: zodResolver(tripFormSchema),
    defaultValues: {
      tripMode: initialTripMode,
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

  const {
    register,
    getValues,
    setValue,
    trigger,
    watch,
    formState: { errors }
  } = form;

  const createMutation = useMutation({
    mutationFn: async ({ input, generate }: { input: CreateTripInput; generate: boolean }) => {
      const trip = await createTrip(input);
      if (!generate) {
        return { trip, generationJobId: null, generationStartFailed: false };
      }
      try {
        const job = await createGenerationJob(trip.id, {
          jobType: "full_generation",
          expectedItineraryRevision: trip.itineraryRevision,
          instruction: undefined
        });
        return { trip, generationJobId: job.id, generationStartFailed: false };
      } catch {
        // The trip itself was created successfully. Let the detail page offer the
        // standard retry action instead of stranding the user on this form.
        return { trip, generationJobId: null, generationStartFailed: true };
      }
    },
    onSuccess: async ({ trip, generationJobId, generationStartFailed }) => {
      onboarding.markFirstTripCreated(trip.createdAt);
      await queryClient.invalidateQueries({ queryKey: tripKeys.lists() });
      const query = new URLSearchParams();
      if (generationJobId) query.set("generationJob", generationJobId);
      if (generationStartFailed) query.set("generationStart", "failed");
      router.push(`/trips/${trip.id}${query.size ? `?${query.toString()}` : ""}`);
    }
  });

  useEffect(() => {
    register("interests");
  }, [register]);

  useEffect(() => {
    setValue("tripMode", initialTripMode);
  }, [initialTripMode, setValue]);

  useEffect(() => {
    if (
      preferenceDefaultsApplied.current ||
      !profileQuery.data ||
      !preferencesQuery.data ||
      form.formState.isDirty
    ) {
      return;
    }
    preferenceDefaultsApplied.current = true;
    const profile = profileQuery.data;
    const preferences = preferencesQuery.data;
    const interests = preferences.travelStyles.filter((style) =>
      interestOptions.some((option) => option.value === style)
    );
    const pace = preferences.pace === "intensive" ? "packed" : preferences.pace;
    setValue("budgetCurrency", profile.preferredCurrency || "EUR");
    setValue("interests", interests);
    setValue("pace", pace);
    setAdvancedPreferences((current) => ({
      ...current,
      outputLanguage: profile.preferredLanguage,
      pace,
      maxWalkingKmPerDay: preferences.maxWalkingKmPerDay,
      preferredModes: preferences.preferredTransport as TransportMode[],
      tripStyles: preferences.travelStyles.filter(isTripStyle) as TripStyle[],
      avoid: preferences.avoid.join(", ")
    }));
    if (profile.homeCity) {
      setRoute((current) => ({
        ...current,
        origin: { name: profile.homeCity, country: profile.homeCountry }
      }));
    }
  }, [form.formState.isDirty, preferencesQuery.data, profileQuery.data, setValue]);

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
  const selectedPace = watch("pace") ?? "balanced";
  const selectedWorkspaceId = watch("workspaceId") ?? "";
  const selectedStartDate = watch("startDate") ?? "";
  const selectedBudgetAmount = watch("budgetAmount");
  const selectedTravelers = watch("travelers") ?? 1;
  const selectedDestination = watch("destination") ?? "";

  function toggleInterest(value: string) {
    const next = selectedInterests.includes(value)
      ? selectedInterests.filter((item) => item !== value)
      : [...selectedInterests, value];
    setValue("interests", next, { shouldDirty: true });
  }

  async function continueToNextStep() {
    const fields = fieldsForStep(currentStep);
    const valid = fields.length === 0 ? true : await trigger(fields);
    if (!valid) return;
    if (currentStep === 4 && !validateRoute()) return;
    setCurrentStep((step) => Math.min(5, step + 1) as CreateTripStep);
  }

  function validateRoute() {
    if (selectedTripMode !== "multi_destination") {
      setRouteError(null);
      return true;
    }
    const warning = getRouteValidationWarnings(routeWithAdvancedPreferences(route), selectedDays).find(
      (item) => item.severity === "error"
    );
    if (warning) {
      setRouteError(warning.message);
      return false;
    }
    setRouteError(null);
    return true;
  }

  async function submitTrip(generate: boolean) {
    const valid = await trigger();
    if (!valid || !validateRoute()) return;
    const values = getValues();
    const routeForSubmit = values.tripMode === "multi_destination" ? routeWithAdvancedPreferences(route) : null;
    const destination =
      values.tripMode === "multi_destination"
        ? values.destination.trim() || deriveRouteDestination(route)
        : values.destination;
    createMutation.mutate({
      generate,
      input: {
        destination,
        tripType: values.tripMode,
        route: routeForSubmit,
        workspaceId: values.workspaceId || undefined,
        startDate: values.startDate,
        days: values.days,
        budgetAmount: values.budgetAmount,
        budgetCurrency: values.budgetCurrency.toUpperCase(),
        travelers: values.travelers,
        interests: values.interests ?? [],
        pace: values.pace
      }
    });
  }

  function previewPlanningConstraints() {
    planningPreviewMutation.mutate(buildPlanningPreviewRequest(getValues()));
  }

  function buildPlanningPreviewRequest(values: TripFormValues): PlanningConstraintsPreviewRequest {
    const routeForPreview = values.tripMode === "multi_destination" ? routeWithAdvancedPreferences(route) : null;
    const preferredModes = advancedPreferences.preferredModes ?? routeForPreview?.preferences?.preferredModes ?? [];
    const avoidModes = advancedPreferences.avoidModes ?? routeForPreview?.preferences?.avoidModes ?? [];
    const tripStyles = advancedPreferences.tripStyles?.length
      ? advancedPreferences.tripStyles
      : routeForPreview?.preferences?.tripStyles ?? [];
    return {
      source: "trip_generation",
      workspaceId: values.workspaceId || null,
      request: {
        tripType: values.tripMode,
        destination: values.tripMode === "multi_destination" ? values.destination.trim() || deriveRouteDestination(route) : values.destination,
        outputLanguage: advancedPreferences.outputLanguage,
        startDate: values.startDate || undefined,
        durationDays: values.days,
        budget: {
          amount: optionalNumber(values.budgetAmount) ?? null,
          currency: values.budgetCurrency.toUpperCase(),
          strictness: advancedPreferences.budgetStrictness ?? "target"
        },
        travelers: { count: values.travelers },
        pace: advancedPreferences.pace ?? values.pace,
        walking: {
          maxKmPerDay: advancedPreferences.maxWalkingKmPerDay ?? null,
          allowLongHikes: tripStyles.includes("hiking")
        },
        transport: {
          preferredModes,
          avoidModes,
          carAvailable: advancedPreferences.carAvailable ?? false,
          maxTransferHoursPerDay: advancedPreferences.maxTransferHoursPerDay ?? null
        },
        route: routeForPreview,
        tripStyles,
        interests: values.interests ?? [],
        avoid: splitPreferenceList(advancedPreferences.avoid),
        mustHave: splitPreferenceList(advancedPreferences.mustHave)
      },
      includePreviousTripSignals: true,
      includeWorkspacePolicy: true,
      includeRoute: true
    };
  }

  function routeWithAdvancedPreferences(input: TripRoute): TripRoute {
    const tripStyles = advancedPreferences.tripStyles?.length
      ? advancedPreferences.tripStyles
      : input.preferences?.tripStyles;
    return {
      ...input,
      preferences: {
        ...input.preferences,
        ...(advancedPreferences.preferredModes ? { preferredModes: advancedPreferences.preferredModes } : {}),
        ...(advancedPreferences.avoidModes ? { avoidModes: advancedPreferences.avoidModes } : {}),
        carAvailable: advancedPreferences.carAvailable ?? input.preferences?.carAvailable ?? false,
        maxTransferHoursPerDay: advancedPreferences.maxTransferHoursPerDay ?? input.preferences?.maxTransferHoursPerDay ?? null,
        ...(tripStyles ? { tripStyles } : {})
      }
    };
  }

  const workspaceWarnings = [
    ...(planningPreviewMutation.data?.blockers ?? []),
    ...(planningPreviewMutation.data?.warnings ?? [])
  ].map((issue) => issue.message);

  return (
    <form
      className="flex flex-col gap-7 rounded-[22px] border border-sand-300 bg-white px-6 py-7 shadow-[0_1px_2px_rgba(34,26,20,0.04),0_14px_36px_rgba(34,26,20,0.06)] sm:px-10 sm:py-9"
      onSubmit={(event) => event.preventDefault()}
    >
      <CreateTripStepper
        currentStep={currentStep}
        labels={[
          creationT("whereWhen"),
          creationT("who"),
          creationT("budgetStyle"),
          creationT("routeTransport"),
          creationT("reviewGenerate")
        ]}
      />
      {completenessQuery.data && completenessQuery.data.score < 70 ? <PreferenceReviewPrompt compact /> : null}

      {currentStep === 1 ? (
        <section>
          <StepHeader title="Where and when?" description="Start with the essentials. You can add preferences and route details later." />
          <div className="mt-5 flex flex-col gap-[18px]">
            <div className="inline-flex w-fit gap-1 rounded-full border border-sand-300 bg-sand-50 p-1">
              {[
                { value: "single_destination", label: "Single destination" },
                { value: "multi_destination", label: "Multi-destination route" }
              ].map((option) => (
                <button key={option.value} type="button" aria-pressed={selectedTripMode === option.value} onClick={() => setValue("tripMode", option.value as TripFormValues["tripMode"], { shouldDirty: true })} className={selectedTripMode === option.value ? "h-10 rounded-full bg-cocoa-900 px-5 text-[14px] font-semibold text-sand-150" : "h-10 rounded-full px-5 text-[14px] font-medium text-cocoa-500 transition hover:bg-sand-200 hover:text-cocoa-900"}>
                  {option.label}
                </button>
              ))}
            </div>
            <label className="block">
              <span className={FIELD_LABEL}>{selectedTripMode === "multi_destination" ? "Trip name or region" : "Destination"}</span>
              <span className="mt-2 flex h-[52px] items-center gap-2.5 rounded-[14px] border border-sand-400 bg-[#FFFDFA] px-4 transition focus-within:border-clay focus-within:ring-[3px] focus-within:ring-clay-tint">
                <MapPinIcon className="h-[19px] w-[19px] shrink-0 text-clay" />
                <input id="destination" placeholder={selectedTripMode === "multi_destination" ? "Optional route name" : "City, region, or country"} aria-invalid={Boolean(errors.destination)} {...register("destination")} className="flex-1 border-none bg-transparent text-[16px] text-cocoa-900 outline-none placeholder:text-cocoa-400" />
              </span>
              {errors.destination?.message ? <span className={FIELD_ERROR}>{errors.destination.message}</span> : null}
            </label>
            <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-3">
              <Field label="Start date" error={errors.startDate?.message}><input id="startDate" type="date" className={FIELD_INPUT} {...register("startDate")} /></Field>
              <Field label="Days" error={errors.days?.message}><input id="days" type="number" min={1} max={30} className={FIELD_INPUT} {...register("days", { valueAsNumber: true })} /></Field>
              <label className="block"><span className={FIELD_LABEL}>Trip scope</span><select id="workspaceId" className={FIELD_INPUT} {...register("workspaceId")}><option value="">Personal trip</option>{editableWorkspaces.map((workspace) => <option key={workspace.id} value={workspace.id}>{workspace.name}</option>)}</select></label>
            </div>
          </div>
        </section>
      ) : null}

      {currentStep === 2 ? (
        <section>
          <StepHeader title="Who is going?" description="Set the group size. Individual preferences stay optional and can be refined later." />
          <div className="mt-5 max-w-xs"><Field label="Travelers" error={errors.travelers?.message}><input id="travelers" type="number" min={1} className={FIELD_INPUT} {...register("travelers", { valueAsNumber: true })} /></Field></div>
        </section>
      ) : null}

      {currentStep === 3 ? (
        <section>
          <StepHeader title="Budget and style" description="Use your saved preferences as a starting point, then adjust only what matters for this trip." />
          <div className="mt-5 grid grid-cols-1 gap-3.5 sm:grid-cols-2">
            <Field label="Budget (optional)" error={errors.budgetAmount?.message}><input id="budgetAmount" type="number" min={0} step="0.01" placeholder="600" className={FIELD_INPUT} {...register("budgetAmount", { setValueAs: (value) => (value === "" ? undefined : Number(value)) })} /></Field>
            <label className="block"><span className={FIELD_LABEL}>Currency</span><select id="budgetCurrency" className={FIELD_INPUT} {...register("budgetCurrency")}><option value="EUR">EUR</option><option value="USD">USD</option><option value="GBP">GBP</option><option value="CZK">CZK</option></select></label>
          </div>
          <p className="mt-6 text-[13.5px] font-semibold text-cocoa-700">Pace</p>
          <div className="mt-2 inline-flex gap-1 rounded-full border border-sand-300 bg-sand-50 p-1">{paceOptions.map((option) => <label key={option.value} className="cursor-pointer"><input type="radio" value={option.value} className="peer sr-only" {...register("pace")} /><span className="flex h-[38px] items-center rounded-full px-5 text-[14px] font-medium text-cocoa-500 transition hover:bg-sand-200 peer-checked:bg-cocoa-900 peer-checked:font-semibold peer-checked:text-sand-150 peer-checked:hover:bg-cocoa-900">{option.label}</span></label>)}</div>
          <p className="mt-6 text-[13.5px] font-semibold text-cocoa-700">Interests</p>
          <div className="mt-2.5 flex flex-wrap gap-2">{interestOptions.map((option) => { const selected = selectedInterests.includes(option.value); return <button key={option.value} type="button" aria-pressed={selected} onClick={() => toggleInterest(option.value)} className={selected ? "inline-flex h-10 items-center gap-1.5 rounded-full border border-clay bg-clay-tint px-[18px] text-[14px] font-semibold text-clay-deep transition" : "inline-flex h-10 items-center rounded-full border border-sand-400 bg-white px-[18px] text-[14px] font-medium text-cocoa-500 transition hover:border-sand-600 hover:text-cocoa-900"}>{selected ? <CheckCircleIcon className="h-[13px] w-[13px]" /> : null}{option.label}</button>; })}</div>
        </section>
      ) : null}

      {currentStep === 4 ? (
        <section>
          <StepHeader title="Route and transport" description="Optional for a single destination. Add route stops and transport preferences only when they help the plan." />
          {selectedTripMode === "multi_destination" ? (
            <div className="mt-5">
              <TripRouteBuilder value={route} onChange={setRoute} totalDays={selectedDays} currency={selectedCurrency} />
              {routeError ? <p className={FIELD_ERROR}>{routeError}</p> : null}
              <details className="mt-5 rounded-xl border border-sand-300 bg-sand-50 p-4"><summary className="cursor-pointer text-[14px] font-semibold text-cocoa-800">Find route alternatives</summary><div className="mt-4"><RouteAlternativesPanel canCreateTrip defaultPrompt={`Plan a ${selectedDays}-day route with ${selectedInterests.join(", ") || "balanced travel"} from ${route.origin?.name || "my origin"}.`} preTripDefaults={{ origin: route.origin, durationDays: selectedDays, startDate: selectedStartDate || undefined, budget: optionalNumber(selectedBudgetAmount) != null ? { amount: optionalNumber(selectedBudgetAmount), currency: selectedCurrency } : undefined, travelers: selectedTravelers, workspaceId: selectedWorkspaceId || undefined, transport: { preferredModes: advancedPreferences.preferredModes, avoidModes: advancedPreferences.avoidModes, carAvailable: advancedPreferences.carAvailable ?? false, maxTransferHoursPerDay: advancedPreferences.maxTransferHoursPerDay ?? undefined }, tripStyles: advancedPreferences.tripStyles, outputLanguage: advancedPreferences.outputLanguage ?? "en" }} onTripCreated={(trip) => { onboarding.markFirstTripCreated(trip.createdAt); router.push(`/trips/${trip.id}`); }} /></div></details>
            </div>
          ) : <p className="mt-5 rounded-xl border border-sand-300 bg-sand-50 p-4 text-[14px] leading-6 text-cocoa-500">We&apos;ll plan within {selectedDestination || "your destination"}. You can add transport or convert to a multi-destination route later.</p>}
          <details className="mt-5 rounded-xl border border-sand-300 bg-sand-50 p-4"><summary className="cursor-pointer text-[14px] font-semibold text-cocoa-800">{creationT("advancedOptions")}</summary><div className="mt-5"><AdvancedTripPreferencesForm value={{ ...advancedPreferences, pace: advancedPreferences.pace ?? selectedPace }} onChange={setAdvancedPreferences} /></div><div className="mt-5 rounded-xl border border-sand-300 bg-white p-4"><PlanningConstraintsPreviewPanel error={planningPreviewMutation.isError ? getErrorMessage(planningPreviewMutation.error, "Could not preview AI settings.") : null} isLoading={planningPreviewMutation.isPending} onPreview={previewPlanningConstraints} preview={planningPreviewMutation.data ?? null} /></div></details>
        </section>
      ) : null}

      {currentStep === 5 ? <CreateTripReviewStep destination={selectedDestination} days={selectedDays} startDate={selectedStartDate} travelers={selectedTravelers} budgetAmount={selectedBudgetAmount} budgetCurrency={selectedCurrency} interests={selectedInterests} pace={selectedPace} outputLanguage={advancedPreferences.outputLanguage ?? "en"} route={selectedTripMode === "multi_destination" ? route : null} workspaceWarnings={workspaceWarnings} /> : null}

      {createMutation.isError ? <div role="alert" className="rounded-2xl border border-clay/30 bg-clay-tint/50 px-4 py-3 text-[14px] text-clay-deep">{getErrorMessage(createMutation.error, "Could not create trip.")}</div> : null}

      <div className="flex flex-col-reverse gap-3 border-t border-sand-200 pt-6 sm:flex-row sm:items-center sm:justify-between">
        <button type="button" disabled={createMutation.isPending} onClick={() => currentStep === 1 ? router.push("/trips") : setCurrentStep((step) => Math.max(1, step - 1) as CreateTripStep)} className="inline-flex h-12 items-center justify-center rounded-full px-[22px] text-[15px] font-medium text-cocoa-500 transition hover:bg-sand-200 hover:text-cocoa-900 disabled:cursor-not-allowed disabled:opacity-60">{currentStep === 1 ? "Cancel" : "Back"}</button>
        {currentStep < 5 ? <button type="button" onClick={continueToNextStep} className="inline-flex h-12 items-center justify-center rounded-full bg-cocoa-900 px-[26px] text-[15px] font-semibold text-sand-100 transition hover:bg-cocoa-800">Continue</button> : <div className="flex flex-col gap-2 sm:flex-row"><button type="button" disabled={createMutation.isPending} onClick={() => void submitTrip(false)} className="inline-flex h-12 items-center justify-center rounded-full border border-sand-500 bg-white px-5 text-[14px] font-semibold text-cocoa-700 transition hover:border-sand-700 disabled:cursor-not-allowed disabled:opacity-60">{creationT("createWithoutGenerating")}</button><button type="button" disabled={createMutation.isPending} onClick={() => void submitTrip(true)} className="inline-flex h-12 items-center justify-center gap-2.5 rounded-full bg-clay px-[22px] text-[14px] font-semibold text-sand-100 shadow-[0_8px_20px_rgba(192,91,59,0.25)] transition hover:bg-clay-dark disabled:cursor-not-allowed disabled:opacity-60"><SparklesIcon className="h-[17px] w-[17px]" />{createMutation.isPending ? "Creating…" : creationT("createAndGenerate")}</button></div>}
      </div>
    </form>
  );
}

function StepHeader({ title, description }: { title: string; description: string }) {
  return <><h2 className="font-newsreader text-[24px] font-semibold text-cocoa-900">{title}</h2><p className="mt-2 max-w-2xl text-[14px] leading-6 text-cocoa-500">{description}</p></>;
}

function Field({ label, error, children }: { label: string; error?: string; children: React.ReactNode }) {
  return <label className="block"><span className={FIELD_LABEL}>{label}</span>{children}{error ? <span className={FIELD_ERROR}>{error}</span> : null}</label>;
}

function fieldsForStep(step: CreateTripStep) {
  switch (step) {
    case 1: return ["tripMode", "destination", "startDate", "days", "workspaceId"] as const;
    case 2: return ["travelers"] as const;
    case 3: return ["budgetAmount", "budgetCurrency", "pace", "interests"] as const;
    default: return [] as const;
  }
}

function optionalNumber(value: number | undefined) { return typeof value === "number" && Number.isFinite(value) ? value : undefined; }

function splitPreferenceList(value?: string) { return (value ?? "").split(/[\n,]/).map((item) => item.trim()).filter(Boolean); }

function deriveRouteDestination(route: ReturnType<typeof createDefaultTripRoute>) {
  const firstStop = route.stops.find((stop) => stop.destination.trim().length > 0);
  if (!firstStop) return "Multi-destination route";
  const sameCountry = firstStop.country && route.stops.every((stop) => (stop.country ?? "").trim() === firstStop.country?.trim());
  return sameCountry ? `${firstStop.country} route` : `${firstStop.destination} route`;
}

function isTripStyle(value: string): value is TripStyle {
  return ["city_break", "road_trip", "train_trip", "backpacking", "camping", "hiking", "island_hopping", "nature", "beach", "food", "culture", "adventure", "family", "romantic", "low_budget", "luxury", "hidden_gem"].includes(value);
}
