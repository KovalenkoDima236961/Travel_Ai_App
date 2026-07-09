import { ComponentType } from "react";
import { CubeTransparentIcon, PencilSquareIcon, SparklesIcon } from "./icons";

type Step = {
  label: string;
  title: string;
  body: string;
  Icon: ComponentType<{ className?: string }>;
};

const steps: Step[] = [
  {
    label: "STEP ONE",
    title: "Describe your trip",
    body: "Destination, dates, budget, travelers, and the pace and interests that fit you.",
    Icon: PencilSquareIcon
  },
  {
    label: "STEP TWO",
    title: "AI drafts the itinerary",
    body: "Real places with routes, opening hours, weather context, and cost estimates.",
    Icon: SparklesIcon
  },
  {
    label: "STEP THREE",
    title: "Refine and go",
    body: "Edit every day, share with companions, split costs, and take it offline.",
    Icon: CubeTransparentIcon
  }
];

export function HowItWorks() {
  return (
    <section className="border-t border-sand-300 bg-white">
      <div className="mx-auto max-w-[1240px] px-6 py-[76px] sm:px-10">
        <p className="text-center text-[12.5px] font-semibold uppercase tracking-[0.14em] text-clay">
          How it works
        </p>
        <h2 className="mx-auto mt-3.5 max-w-[560px] text-balance text-center font-newsreader text-[32px] font-medium leading-[1.15] tracking-[-0.015em] text-cocoa-900 sm:text-[38px]">
          From idea to itinerary in three steps
        </h2>
        <div className="mt-14 grid gap-10 sm:grid-cols-3">
          {steps.map(({ label, title, body, Icon }) => (
            <div key={label} className="text-center">
              <div className="mx-auto flex h-[60px] w-[60px] items-center justify-center rounded-full bg-clay-tint text-clay-dark">
                <Icon className="h-[25px] w-[25px]" />
              </div>
              <p className="mt-[22px] text-[13px] font-semibold tracking-[0.1em] text-sand-600">
                {label}
              </p>
              <h3 className="mt-2 font-newsreader text-[23px] font-semibold text-cocoa-900">
                {title}
              </h3>
              <p className="mx-auto mt-2.5 max-w-[280px] text-[15px] leading-[1.6] text-cocoa-500">
                {body}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
