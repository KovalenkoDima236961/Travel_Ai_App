"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";

const ACTIONS = [
  "cheaper",
  "warmer",
  "moreNature",
  "moreCity",
  "lessWalking",
  "differentCountry",
  "similarPlaces",
  "hiddenGem",
  "betterFood",
  "betterMuseums"
] as const;

export function TripDiscoveryRefineBar({
  isPending,
  onRefine
}: {
  isPending: boolean;
  onRefine: (instruction: string) => void;
}) {
  const t = useTranslations("tripDiscovery");
  const [instruction, setInstruction] = useState("");
  return (
    <section className="rounded-[20px] border border-sand-300 bg-[#F4EDE4] px-6 py-6">
      <h3 className="font-newsreader text-[21px] font-semibold text-cocoa-900">
        {t("refineTitle")}
      </h3>
      <p className="mt-1.5 text-[13.5px] text-cocoa-500">{t("refineSubtitle")}</p>
      <div className="mt-4 flex flex-wrap gap-2">
        {ACTIONS.map((action) => (
          <button
            key={action}
            type="button"
            disabled={isPending}
            onClick={() => onRefine(t(`refineInstructions.${action}`))}
            className="rounded-full border border-sand-400 bg-white px-3.5 py-2 text-[12.5px] font-semibold text-cocoa-600 hover:border-clay/50 hover:text-clay-deep disabled:opacity-50"
          >
            {t(`refine.${action}`)}
          </button>
        ))}
      </div>
      <div className="mt-4 flex flex-col gap-2 sm:flex-row">
        <label htmlFor="trip-discovery-refine" className="sr-only">
          {t("refinePlaceholder")}
        </label>
        <input
          id="trip-discovery-refine"
          value={instruction}
          maxLength={1000}
          onChange={(event) => setInstruction(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && instruction.trim()) {
              onRefine(instruction.trim());
              setInstruction("");
            }
          }}
          placeholder={t("refinePlaceholder")}
          className="h-11 flex-1 rounded-full border border-sand-400 bg-white px-4 text-[13.5px] text-cocoa-800 outline-none focus:border-clay"
        />
        <button
          type="button"
          disabled={isPending || !instruction.trim()}
          onClick={() => {
            onRefine(instruction.trim());
            setInstruction("");
          }}
          className="h-11 rounded-full bg-cocoa-900 px-5 text-[13px] font-semibold text-sand-100 disabled:opacity-50"
        >
          {isPending ? t("refining") : t("refineButton")}
        </button>
      </div>
    </section>
  );
}
