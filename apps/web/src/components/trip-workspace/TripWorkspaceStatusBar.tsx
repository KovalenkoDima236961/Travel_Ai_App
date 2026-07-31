"use client";

import { useTranslations } from "next-intl";

export function TripWorkspaceStatusBar({
  archived,
  offline,
  readOnly,
  cachedAt
}: {
  archived: boolean;
  offline: boolean;
  readOnly: boolean;
  cachedAt?: string | null;
}) {
  const t = useTranslations("tripWorkspace.status");
  if (!archived && !offline && !readOnly) {
    return null;
  }
  return (
    <div aria-label={t("label")} className="mt-3 flex flex-wrap gap-2" role="status">
      {archived ? <StatusChip title={t("archivedTitle")} description={t("archivedDescription")} /> : null}
      {!archived && readOnly ? <StatusChip title={t("readOnlyTitle")} description={t("readOnlyDescription")} /> : null}
      {offline ? (
        <StatusChip
          title={t("offlineTitle")}
          description={cachedAt ? t("offlineCachedDescription", { cachedAt: formatCachedAt(cachedAt) }) : t("offlineDescription")}
        />
      ) : null}
    </div>
  );
}

function StatusChip({ title, description }: { title: string; description: string }) {
  return (
    <div className="min-w-0 flex-1 rounded-xl border border-[#EAD9B8] bg-[#FDF7E8] px-3 py-2 text-sm text-[#6E4B1E]">
      <span className="font-semibold">{title}</span>
      <span className="ml-1">{description}</span>
    </div>
  );
}

function formatCachedAt(value: string) {
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString();
}
