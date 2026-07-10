"use client";

import { useTranslations } from "next-intl";
import type { TripDiscoverySession } from "@/types/trip-discovery";

export function DiscoverySessionHistory({
  sessions,
  onSelect
}: {
  sessions: TripDiscoverySession[];
  onSelect: (session: TripDiscoverySession) => void;
}) {
  const t = useTranslations("tripDiscovery");
  if (sessions.length === 0) return null;
  return (
    <details className="rounded-[18px] border border-sand-300 bg-white px-5 py-4">
      <summary className="cursor-pointer text-[13.5px] font-semibold text-cocoa-700">
        {t("recentDiscoveries")}
      </summary>
      <div className="mt-3 space-y-2">
        {sessions.slice(0, 5).map((session) => (
          <button
            key={session.id}
            type="button"
            onClick={() => onSelect(session)}
            className="flex w-full items-center justify-between rounded-xl bg-sand-50 px-3 py-2.5 text-left hover:bg-sand-100"
          >
            <span className="truncate text-[13px] font-medium text-cocoa-700">
              {session.response.sessionTitle}
            </span>
            <span className="ml-3 shrink-0 text-[11px] text-cocoa-400">
              {new Date(session.createdAt).toLocaleDateString()}
            </span>
          </button>
        ))}
      </div>
    </details>
  );
}
