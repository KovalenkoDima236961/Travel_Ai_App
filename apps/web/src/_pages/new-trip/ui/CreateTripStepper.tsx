import { cn } from "@/shared/lib/cn";

export type CreateTripStep = 1 | 2 | 3 | 4 | 5;

const STEPS: Array<{ number: CreateTripStep; title: string }> = [
  { number: 1, title: "Where and when?" },
  { number: 2, title: "Who is going?" },
  { number: 3, title: "Budget and style" },
  { number: 4, title: "Route and transport" },
  { number: 5, title: "Review and generate" }
];

export function CreateTripStepper({
  currentStep,
  labels
}: {
  currentStep: CreateTripStep;
  labels?: string[];
}) {
  return (
    <ol aria-label="Create trip progress" className="grid gap-2 sm:grid-cols-5 sm:gap-3">
      {STEPS.map((step, index) => {
        const active = step.number === currentStep;
        const complete = step.number < currentStep;
        return (
          <li
            key={step.number}
            className={cn(
              "flex min-h-12 items-center gap-2 rounded-xl border px-3 py-2 text-[12.5px] font-semibold transition",
              active
                ? "border-clay bg-clay-tint text-clay-deep"
                : complete
                  ? "border-[#CFE1D2] bg-[#F2F7F1] text-[#38543F]"
                  : "border-sand-300 bg-sand-50 text-cocoa-400"
            )}
          >
            <span
              aria-hidden="true"
              className={cn(
                "flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-[11px]",
                active
                  ? "bg-clay text-sand-100"
                  : complete
                    ? "bg-[#3E6B5A] text-white"
                    : "bg-sand-200 text-cocoa-500"
              )}
            >
              {complete ? "✓" : step.number}
            </span>
            <span>{labels?.[index] ?? step.title}</span>
          </li>
        );
      })}
    </ol>
  );
}
