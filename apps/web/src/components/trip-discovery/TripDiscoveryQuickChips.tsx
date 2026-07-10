"use client";

import { useTranslations } from "next-intl";

export const DISCOVERY_CHIPS = [
  "weekend",
  "warm",
  "mountains",
  "food",
  "museums",
  "lowBudget",
  "noFlights",
  "hiddenGem",
  "nature",
  "cityBreak",
  "romantic",
  "family",
  "lessWalking"
] as const;

export function TripDiscoveryQuickChips({
  selected,
  onToggle
}: {
  selected: string[];
  onToggle: (chip: string) => void;
}) {
  const t = useTranslations("tripDiscovery");
  return (
    <div className="flex flex-wrap gap-2" aria-label={t("quickPreferences")}>
      {DISCOVERY_CHIPS.map((chip) => {
        const active = selected.includes(chip);
        return (
          <button
            key={chip}
            type="button"
            aria-pressed={active}
            onClick={() => onToggle(chip)}
            className={
              active
                ? "rounded-full border border-clay bg-clay-tint px-4 py-2 text-[13px] font-semibold text-clay-deep"
                : "rounded-full border border-sand-400 bg-white px-4 py-2 text-[13px] font-medium text-cocoa-500 transition hover:border-clay/60 hover:text-cocoa-900"
            }
          >
            {t(`chips.${chip}`)}
          </button>
        );
      })}
    </div>
  );
}
