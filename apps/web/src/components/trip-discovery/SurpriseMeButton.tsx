"use client";

import { useTranslations } from "next-intl";

export function SurpriseMeButton({
  isPending,
  onClick
}: {
  isPending: boolean;
  onClick: () => void;
}) {
  const t = useTranslations("tripDiscovery");
  return (
    <button
      type="button"
      disabled={isPending}
      onClick={onClick}
      className="inline-flex h-12 items-center justify-center gap-2 rounded-full border border-clay/40 bg-clay-tint px-6 text-[14px] font-semibold text-clay-deep transition hover:border-clay hover:bg-[#F5DED2] disabled:cursor-not-allowed disabled:opacity-60"
    >
      <span aria-hidden="true">✦</span>
      {isPending ? t("findingPlaces") : t("surpriseMe")}
    </button>
  );
}
