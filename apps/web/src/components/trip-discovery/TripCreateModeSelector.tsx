"use client";

import { useTranslations } from "next-intl";

export type TripCreateMode = "known" | "discover";

export function TripCreateModeSelector({
  value,
  onChange
}: {
  value: TripCreateMode;
  onChange: (value: TripCreateMode) => void;
}) {
  const t = useTranslations("tripDiscovery");
  return (
    <div
      className="grid gap-2 rounded-[18px] border border-sand-300 bg-white p-2 sm:grid-cols-2"
      role="radiogroup"
      aria-label={t("modeLabel")}
    >
      {(["known", "discover"] as const).map((mode) => {
        const selected = value === mode;
        return (
          <button
            key={mode}
            type="button"
            role="radio"
            aria-checked={selected}
            onClick={() => onChange(mode)}
            className={
              selected
                ? "rounded-[13px] bg-cocoa-900 px-5 py-4 text-left text-sand-100 shadow-sm"
                : "rounded-[13px] px-5 py-4 text-left text-cocoa-600 transition hover:bg-sand-100"
            }
          >
            <span className="block text-[14.5px] font-semibold">
              {mode === "known" ? t("knownDestination") : t("helpMeChoose")}
            </span>
            <span className={`mt-1 block text-[12.5px] ${selected ? "text-sand-300" : "text-cocoa-400"}`}>
              {mode === "known" ? t("knownDestinationHint") : t("helpMeChooseHint")}
            </span>
          </button>
        );
      })}
    </div>
  );
}
