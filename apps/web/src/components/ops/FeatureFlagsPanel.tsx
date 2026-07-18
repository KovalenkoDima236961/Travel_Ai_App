"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import {
  getOpsFeatureFlagAudit,
  getOpsFeatureFlags,
  opsKeys,
  resetOpsFeatureFlag,
  updateOpsFeatureFlag,
  type OpsFeatureFlag
} from "@/lib/api/ops";

const highRisk = new Set([
  "public_sharing_enabled",
  "data_exports_enabled",
  "real_providers_enabled",
  "calendar_sync_enabled"
]);

export function FeatureFlagsPanel() {
  const t = useTranslations("featureFlags");
  const queryClient = useQueryClient();
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("");
  const flags = useQuery({
    queryKey: opsKeys.featureFlags,
    queryFn: getOpsFeatureFlags,
    staleTime: 30_000
  });
  const audit = useQuery({
    queryKey: opsKeys.featureFlagAudit(selectedKey ?? ""),
    queryFn: () => getOpsFeatureFlagAudit(selectedKey ?? ""),
    enabled: Boolean(selectedKey)
  });
  const refresh = () => void queryClient.invalidateQueries({ queryKey: opsKeys.featureFlags });
  const update = useMutation({
    mutationFn: ({ key, value, reason }: { key: string; value: boolean; reason: string }) =>
      updateOpsFeatureFlag(key, value, reason),
    onSuccess: refresh
  });
  const reset = useMutation({
    mutationFn: ({ key, reason }: { key: string; reason: string }) =>
      resetOpsFeatureFlag(key, reason),
    onSuccess: refresh
  });
  const categories = useMemo(
    () => [...new Set((flags.data?.flags ?? []).map((flag) => flag.category))].sort(),
    [flags.data?.flags]
  );
  const visibleFlags = useMemo(() => {
    const normalizedSearch = search.trim().toLowerCase();
    return (flags.data?.flags ?? []).filter((flag) =>
      (!category || flag.category === category) &&
      (!normalizedSearch || flag.key.includes(normalizedSearch) || flag.description.toLowerCase().includes(normalizedSearch))
    );
  }, [category, flags.data?.flags, search]);

  function reasonFor(flag: OpsFeatureFlag, action: string) {
    const reason = window.prompt(t("reasonPrompt", { action, flag: flag.key }))?.trim();
    return reason ?? "";
  }

  function toggle(flag: OpsFeatureFlag) {
    const next = !flag.value;
    if (highRisk.has(flag.key) && !window.confirm(t("confirmRisk", { flag: flag.key }))) return;
    const reason = reasonFor(flag, next ? t("enabled") : t("disabled"));
    update.mutate({ key: flag.key, value: next, reason });
  }

  function resetToDefault(flag: OpsFeatureFlag) {
    const reason = reasonFor(flag, t("reset"));
    reset.mutate({ key: flag.key, reason });
  }

  return (
    <section className="mt-6 rounded-[20px] border border-sand-300 bg-white p-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">{t("title")}</h2>
          <p className="mt-1 text-sm text-cocoa-500">{t("description")}</p>
        </div>
        <span className="text-xs text-cocoa-400">{flags.data?.environment ?? "—"}</span>
      </div>
      {flags.isError ? <p className="mt-4 text-sm text-red-700">{t("loadError")}</p> : null}
      <div className="mt-4 flex flex-wrap gap-2">
        <input aria-label={t("search")} className="rounded border border-sand-300 px-3 py-2 text-sm" onChange={(event) => setSearch(event.target.value)} placeholder={t("search")} value={search} />
        <select aria-label={t("category")} className="rounded border border-sand-300 px-3 py-2 text-sm" onChange={(event) => setCategory(event.target.value)} value={category}>
          <option value="">{t("allCategories")}</option>
          {categories.map((value) => <option key={value} value={value}>{value}</option>)}
        </select>
      </div>
      <div className="mt-4 overflow-x-auto">
        <table className="min-w-full text-left text-sm">
          <thead className="border-b border-sand-200 text-xs uppercase tracking-wide text-cocoa-400">
            <tr>
              <th className="px-2 py-2">{t("flag")}</th><th className="px-2 py-2">{t("source")}</th>
              <th className="px-2 py-2">{t("enforcement")}</th><th className="px-2 py-2">{t("safe")}</th><th className="px-2 py-2">{t("value")}</th><th className="px-2 py-2" />
            </tr>
          </thead>
          <tbody>{visibleFlags.map((flag) => <FlagRow flag={flag} key={flag.key} onAudit={() => setSelectedKey(flag.key)} onReset={resetToDefault} onToggle={toggle} pending={update.isPending || reset.isPending} />)}{!flags.isLoading && visibleFlags.length === 0 ? <tr><td className="px-2 py-4 text-sm text-cocoa-500" colSpan={6}>{t("noFlags")}</td></tr> : null}</tbody>
        </table>
      </div>
      {selectedKey ? <AuditPanel audit={audit.data?.events ?? []} loading={audit.isLoading} onClose={() => setSelectedKey(null)} title={selectedKey} t={t} /> : null}
    </section>
  );
}

function FlagRow({ flag, onAudit, onReset, onToggle, pending }: { flag: OpsFeatureFlag; onAudit: () => void; onReset: (flag: OpsFeatureFlag) => void; onToggle: (flag: OpsFeatureFlag) => void; pending: boolean }) {
  const t = useTranslations("featureFlags");
  return <tr className="border-b border-sand-100"><td className="px-2 py-3"><button className="font-mono text-xs text-cocoa-900 hover:underline" onClick={onAudit} type="button">{flag.key}</button><p className="mt-1 text-xs text-cocoa-400">{flag.category}</p></td><td className="px-2 py-3 text-cocoa-600">{flag.metadata.source}</td><td className="px-2 py-3 text-cocoa-600">{flag.requiresBackendEnforcement ? t("backend") : t("frontend")}</td><td className="px-2 py-3 text-cocoa-600">{flag.safeForFrontend ? t("yes") : t("no")}</td><td className="px-2 py-3"><span className={flag.value ? "text-emerald-700" : "text-cocoa-500"}>{flag.value ? t("enabled") : t("disabled")}</span></td><td className="px-2 py-3 whitespace-nowrap"><button className="mr-2 rounded border border-sand-300 px-2 py-1 text-xs hover:bg-sand-100" disabled={pending} onClick={() => onToggle(flag)} type="button">{flag.value ? t("disable") : t("enable")}</button><button className="rounded border border-sand-300 px-2 py-1 text-xs hover:bg-sand-100" disabled={pending} onClick={() => onReset(flag)} type="button">{t("reset")}</button></td></tr>;
}

function AuditPanel({ audit, loading, onClose, title, t }: { audit: Awaited<ReturnType<typeof getOpsFeatureFlagAudit>>["events"]; loading: boolean; onClose: () => void; title: string; t: ReturnType<typeof useTranslations> }) {
  return <div className="mt-5 rounded-xl bg-sand-50 p-4"><div className="flex items-center justify-between"><h3 className="font-semibold text-cocoa-900">{t("audit")}: <span className="font-mono text-xs">{title}</span></h3><button className="text-xs text-cocoa-500 hover:underline" onClick={onClose} type="button">{t("close")}</button></div>{loading ? <p className="mt-2 text-sm text-cocoa-500">{t("loading")}</p> : null}{!loading && audit.length === 0 ? <p className="mt-2 text-sm text-cocoa-500">{t("noAudit")}</p> : null}<ul className="mt-2 space-y-2 text-sm">{audit.map((event) => <li key={event.id}><span className="font-medium">{event.action}</span> · {new Date(event.createdAt).toLocaleString()}{event.reason ? ` — ${event.reason}` : ""}</li>)}</ul></div>;
}
