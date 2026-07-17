import type { ReactNode } from "react";

type PreferenceStepProps = {
  title: string;
  description: string;
  children: ReactNode;
};

export function PreferenceStep({ title, description, children }: PreferenceStepProps) {
  return (
    <fieldset className="min-w-0 border-0 p-0">
      <legend className="font-newsreader text-[25px] font-semibold text-cocoa-900">{title}</legend>
      <p className="mt-2 text-[14px] leading-[1.6] text-cocoa-500">{description}</p>
      <div className="mt-6 space-y-5">{children}</div>
    </fieldset>
  );
}
