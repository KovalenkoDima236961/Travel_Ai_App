"use client";

import { useTranslations } from "next-intl";

export function TripDiscoveryHero() {
  const t = useTranslations("tripDiscovery");
  return (
    <section className="relative overflow-hidden rounded-[26px] bg-cocoa-900 px-7 py-9 text-sand-100 shadow-[0_18px_50px_rgba(34,26,20,0.18)] sm:px-10 sm:py-11">
      <div className="absolute -right-12 -top-20 h-52 w-52 rounded-full bg-clay/25 blur-3xl" />
      <div className="absolute -bottom-24 left-1/3 h-48 w-48 rounded-full bg-[#C6A96C]/20 blur-3xl" />
      <div className="relative max-w-[650px]">
        <p className="text-[12px] font-bold uppercase tracking-[0.18em] text-clay-light">
          {t("eyebrow")}
        </p>
        <h2 className="mt-3 font-newsreader text-[34px] font-medium leading-[1.08] tracking-[-0.02em] sm:text-[42px]">
          {t("heroTitle")}
        </h2>
        <p className="mt-4 max-w-[580px] text-[15.5px] leading-7 text-sand-300">
          {t("heroSubtitle")}
        </p>
      </div>
    </section>
  );
}
