"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  alphaKeys,
  createAlphaInvite,
  generateWeeklyAlphaReport,
  getAlphaDashboard,
  inviteFromWaitlist,
  listAlphaFeedback,
  listAlphaInvites,
  listAlphaWaitlist,
  listWeeklyAlphaReports,
  updateAlphaFeedback,
  updateAlphaInvite,
  type AlphaFeedback
} from "@/lib/api/alpha";
import { getErrorMessage } from "@/lib/utils";
import { cn } from "@/shared/lib/cn";
import {
  CARD,
  CARD_HEADING,
  MICRO_LABEL,
  MONO,
  OPS_INPUT,
  OPS_SELECT,
  OUTLINE_BUTTON,
  SMALL_OUTLINE_BUTTON
} from "@/_pages/ops/ui/opsStyles";

const testerGroups = ["external", "internal", "qa", "design_reviewer"] as const;
const feedbackStatuses = ["open", "triaged", "in_progress", "resolved", "closed", "duplicate"];
const feedbackPriorities = ["low", "normal", "high", "urgent"];

export function AlphaProgramPanel() {
  const queryClient = useQueryClient();
  const [testerGroup, setTesterGroup] = useState<(typeof testerGroups)[number]>("external");
  const [maxActivations, setMaxActivations] = useState(1);
  const [expiresAt, setExpiresAt] = useState("");
  const [notes, setNotes] = useState("");
  const [createdCode, setCreatedCode] = useState<string | null>(null);
  const [waitlistStatus, setWaitlistStatus] = useState("");
  const [feedbackStatus, setFeedbackStatus] = useState("open");
  const [reportWeek, setReportWeek] = useState("");

  const dashboard = useQuery({
    queryKey: alphaKeys.dashboard,
    queryFn: getAlphaDashboard,
    staleTime: 30_000
  });
  const invites = useQuery({
    queryKey: alphaKeys.invites,
    queryFn: listAlphaInvites,
    staleTime: 30_000
  });
  const waitlist = useQuery({
    queryKey: alphaKeys.waitlist(waitlistStatus),
    queryFn: () => listAlphaWaitlist(waitlistStatus || undefined),
    staleTime: 30_000
  });
  const feedback = useQuery({
    queryKey: alphaKeys.feedback(feedbackStatus),
    queryFn: () => listAlphaFeedback(feedbackStatus || undefined),
    staleTime: 30_000
  });
  const reports = useQuery({
    queryKey: alphaKeys.weeklyReports,
    queryFn: listWeeklyAlphaReports,
    staleTime: 30_000
  });

  const invalidateAlpha = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: alphaKeys.dashboard }),
      queryClient.invalidateQueries({ queryKey: alphaKeys.invites }),
      queryClient.invalidateQueries({ queryKey: alphaKeys.waitlist(waitlistStatus) }),
      queryClient.invalidateQueries({ queryKey: alphaKeys.feedback(feedbackStatus) }),
      queryClient.invalidateQueries({ queryKey: alphaKeys.weeklyReports })
    ]);

  const createInvite = useMutation({
    mutationFn: () =>
      createAlphaInvite({
        testerGroup,
        maxActivations,
        expiresAt: expiresAt || null,
        notes
      }),
    onSuccess: async (result) => {
      setCreatedCode(result.invite.code ?? result.invite.codeDisplay);
      setNotes("");
      await invalidateAlpha();
    }
  });

  const toggleInvite = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      updateAlphaInvite(id, { enabled }),
    onSuccess: invalidateAlpha
  });

  const inviteWaitlist = useMutation({
    mutationFn: (waitlistId: string) =>
      inviteFromWaitlist({
        waitlistId,
        testerGroup,
        expiresAt: expiresAt || null,
        notes: "Ops invite from waitlist"
      }),
    onSuccess: async (result) => {
      setCreatedCode(result.invite.code ?? result.invite.codeDisplay);
      await invalidateAlpha();
    }
  });

  const updateFeedback = useMutation({
    mutationFn: ({
      item,
      patch
    }: {
      item: AlphaFeedback;
      patch: Partial<Pick<AlphaFeedback, "status" | "priority">>;
    }) => updateAlphaFeedback(item.id, patch),
    onSuccess: invalidateAlpha
  });

  const generateReport = useMutation({
    mutationFn: () => generateWeeklyAlphaReport(reportWeek || undefined),
    onSuccess: invalidateAlpha
  });

  const d = dashboard.data;
  const error = [dashboard.error, invites.error, waitlist.error, feedback.error, reports.error].find(Boolean);
  const alertCount = d?.alerts.length ?? 0;
  const unusedFeatures = useMemo(
    () => d?.featureUsage.filter((feature) => feature.unused).map((feature) => feature.feature) ?? [],
    [d?.featureUsage]
  );

  return (
    <section className={cn(CARD, "mt-7")} aria-labelledby="alpha-program-title">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 id="alpha-program-title" className={CARD_HEADING}>Closed alpha program</h2>
        </div>
        <div className="flex flex-wrap gap-2 text-[12.5px]">
          <span className={statusBadge(alertCount > 0 ? "warning" : "ok")}>
            {alertCount > 0 ? `${alertCount} alert${alertCount === 1 ? "" : "s"}` : "Healthy"}
          </span>
          {unusedFeatures.length > 0 ? (
            <span className={statusBadge("neutral")}>{unusedFeatures.length} unused features</span>
          ) : null}
        </div>
      </div>

      {error ? (
        <div className="mt-4 rounded-lg border border-[#E5C3B6] bg-[#FBF0EB] p-3 text-[13px] text-[#B3402E]">
          {getErrorMessage(error, "Alpha dashboard is unavailable.")}
        </div>
      ) : null}

      <div className="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
        <AlphaStat label="Invited" value={d?.users.invited ?? 0} />
        <AlphaStat label="Active" value={d?.users.active ?? 0} tone="ok" />
        <AlphaStat label="Retained" value={d?.users.retained ?? 0} />
        <AlphaStat label="Trips" value={d?.trips.created ?? 0} />
        <AlphaStat label="AI success" value={percent(d?.ai.successRate)} tone="ok" />
        <AlphaStat label="Cost" value={`$${(d?.costs.estimatedOpenAiCost ?? 0).toFixed(2)}`} />
      </div>

      <div className="mt-6 grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
        <div>
          <div className={MICRO_LABEL}>Funnel</div>
          <div className="mt-2 overflow-x-auto rounded-[14px] border border-sand-200">
            <table className="min-w-full text-left text-[13px]">
              <thead className="bg-sand-50 text-[11.5px] uppercase tracking-[0.04em] text-[#A08D78]">
                <tr>
                  <th className="px-3 py-2 font-semibold">Stage</th>
                  <th className="px-3 py-2 font-semibold">Users</th>
                  <th className="px-3 py-2 font-semibold">Conversion</th>
                  <th className="px-3 py-2 font-semibold">Drop-off</th>
                </tr>
              </thead>
              <tbody>
                {(d?.funnel ?? []).map((stage) => (
                  <tr key={stage.name} className="border-t border-sand-200">
                    <td className="px-3 py-2.5 text-cocoa-900">{stage.name}</td>
                    <td className={cn("px-3 py-2.5", MONO)}>{stage.users}</td>
                    <td className={cn("px-3 py-2.5", MONO)}>{percent(stage.conversion)}</td>
                    <td className={cn("px-3 py-2.5", MONO)}>{stage.dropoffFromPrevious}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
        <div>
          <div className={MICRO_LABEL}>AI quality</div>
          <div className="mt-2 grid gap-2 sm:grid-cols-2">
            <AlphaMini label="Generations" value={d?.ai.generations ?? 0} />
            <AlphaMini label="Repair rate" value={percent(d?.ai.repairRate)} />
            <AlphaMini label="Fallback rate" value={percent(d?.ai.fallbackRate)} />
            <AlphaMini label="Avg latency" value={`${d?.ai.averageLatencyMs ?? 0} ms`} />
            <AlphaMini label="Removed places" value={d?.ai.removedPlaces ?? 0} />
            <AlphaMini label="Bad place reports" value={d?.ai.badPlaceReports ?? 0} />
          </div>
        </div>
      </div>

      <div className="mt-6 grid gap-5 xl:grid-cols-2">
        <div>
          <div className="flex flex-wrap items-end justify-between gap-3">
            <div>
              <h3 className="text-[15px] font-semibold text-cocoa-900">Invite management</h3>
              {createdCode ? <p className={cn("mt-1 text-[12.5px] text-cocoa-500", MONO)}>New code: {createdCode}</p> : null}
            </div>
            <button
              className={SMALL_OUTLINE_BUTTON}
              disabled={createInvite.isPending}
              onClick={() => createInvite.mutate()}
              type="button"
            >
              {createInvite.isPending ? "Creating..." : "Create invite"}
            </button>
          </div>
          <div className="mt-3 grid gap-2 sm:grid-cols-4">
            <select className={OPS_SELECT} value={testerGroup} onChange={(event) => setTesterGroup(event.target.value as typeof testerGroup)}>
              {testerGroups.map((group) => <option key={group} value={group}>{group}</option>)}
            </select>
            <input className={OPS_INPUT} min={1} type="number" value={maxActivations} onChange={(event) => setMaxActivations(Number(event.target.value))} />
            <input className={OPS_INPUT} type="date" value={expiresAt} onChange={(event) => setExpiresAt(event.target.value)} />
            <input className={OPS_INPUT} placeholder="Notes" value={notes} onChange={(event) => setNotes(event.target.value)} />
          </div>
          <div className="mt-3 max-h-[260px] overflow-auto rounded-[14px] border border-sand-200">
            {(invites.data?.invites ?? []).map((invite) => (
              <div key={invite.id} className="flex items-center justify-between gap-3 border-t border-sand-200 px-3 py-2.5 first:border-t-0">
                <div>
                  <p className={cn("text-[12.5px] text-cocoa-900", MONO)}>{invite.codeDisplay}</p>
                  <p className="text-[12px] text-cocoa-500">{invite.testerGroup} · {invite.currentActivations}/{invite.maxActivations}</p>
                </div>
                <button
                  className={SMALL_OUTLINE_BUTTON}
                  disabled={toggleInvite.isPending}
                  onClick={() => toggleInvite.mutate({ id: invite.id, enabled: !invite.enabled })}
                  type="button"
                >
                  {invite.enabled ? "Disable" : "Enable"}
                </button>
              </div>
            ))}
          </div>
        </div>

        <div>
          <div className="flex flex-wrap items-end justify-between gap-3">
            <h3 className="text-[15px] font-semibold text-cocoa-900">Waitlist</h3>
            <select className={cn(OPS_SELECT, "max-w-[170px]")} value={waitlistStatus} onChange={(event) => setWaitlistStatus(event.target.value)}>
              <option value="">All</option>
              <option value="registered">Registered</option>
              <option value="invited">Invited</option>
              <option value="accepted">Accepted</option>
              <option value="declined">Declined</option>
              <option value="removed">Removed</option>
            </select>
          </div>
          <div className="mt-3 max-h-[322px] overflow-auto rounded-[14px] border border-sand-200">
            {(waitlist.data?.waitlist ?? []).map((entry) => (
              <div key={entry.id} className="flex items-center justify-between gap-3 border-t border-sand-200 px-3 py-2.5 first:border-t-0">
                <div className="min-w-0">
                  <p className="truncate text-[13px] text-cocoa-900">{entry.email}</p>
                  <p className="text-[12px] text-cocoa-500">{entry.status} · {entry.emailDomain}</p>
                </div>
                <button
                  className={SMALL_OUTLINE_BUTTON}
                  disabled={inviteWaitlist.isPending || entry.status === "removed" || entry.status === "declined"}
                  onClick={() => inviteWaitlist.mutate(entry.id)}
                  type="button"
                >
                  Invite
                </button>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="mt-6 grid gap-5 xl:grid-cols-[1fr_0.9fr]">
        <div>
          <div className="flex flex-wrap items-end justify-between gap-3">
            <h3 className="text-[15px] font-semibold text-cocoa-900">Feedback center</h3>
            <select className={cn(OPS_SELECT, "max-w-[170px]")} value={feedbackStatus} onChange={(event) => setFeedbackStatus(event.target.value)}>
              {feedbackStatuses.map((status) => <option key={status} value={status}>{status}</option>)}
            </select>
          </div>
          <div className="mt-3 max-h-[360px] overflow-auto rounded-[14px] border border-sand-200">
            {(feedback.data?.feedback ?? []).map((item) => (
              <div key={item.id} className="border-t border-sand-200 px-3 py-3 first:border-t-0">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p className="text-[13px] font-semibold text-cocoa-900">{item.title}</p>
                    <p className="mt-1 text-[12px] text-cocoa-500">{item.category} · {item.attachmentCount} attachment metadata</p>
                  </div>
                  <div className="flex gap-2">
                    <select
                      className={cn(OPS_SELECT, "h-[32px] w-[126px] text-[12px]")}
                      value={item.status}
                      onChange={(event) => updateFeedback.mutate({ item, patch: { status: event.target.value as AlphaFeedback["status"] } })}
                    >
                      {feedbackStatuses.map((status) => <option key={status} value={status}>{status}</option>)}
                    </select>
                    <select
                      className={cn(OPS_SELECT, "h-[32px] w-[100px] text-[12px]")}
                      value={item.priority}
                      onChange={(event) => updateFeedback.mutate({ item, patch: { priority: event.target.value as AlphaFeedback["priority"] } })}
                    >
                      {feedbackPriorities.map((priority) => <option key={priority} value={priority}>{priority}</option>)}
                    </select>
                  </div>
                </div>
                <p className="mt-2 line-clamp-2 text-[12.5px] text-cocoa-600">{item.descriptionSanitized}</p>
              </div>
            ))}
          </div>
        </div>

        <div>
          <div className="flex flex-wrap items-end justify-between gap-3">
            <h3 className="text-[15px] font-semibold text-cocoa-900">Weekly reports</h3>
            <button
              className={OUTLINE_BUTTON}
              disabled={generateReport.isPending}
              onClick={() => generateReport.mutate()}
              type="button"
            >
              {generateReport.isPending ? "Generating..." : "Generate"}
            </button>
          </div>
          <input className={cn(OPS_INPUT, "mt-3 max-w-[180px]")} type="date" value={reportWeek} onChange={(event) => setReportWeek(event.target.value)} />
          <div className="mt-3 max-h-[360px] overflow-auto rounded-[14px] border border-sand-200">
            {(reports.data?.reports ?? []).map((report) => (
              <details key={report.id} className="border-t border-sand-200 px-3 py-3 first:border-t-0">
                <summary className="cursor-pointer text-[13px] font-semibold text-cocoa-900">
                  {new Date(report.weekStart).toLocaleDateString()} · generated {new Date(report.generatedAt).toLocaleDateString()}
                </summary>
                <pre className="mt-3 whitespace-pre-wrap rounded-lg bg-sand-50 p-3 text-[12px] leading-5 text-cocoa-700">
                  {report.summaryMarkdown}
                </pre>
              </details>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

function AlphaStat({ label, value, tone = "neutral" }: { label: string; value: string | number; tone?: "ok" | "neutral" }) {
  return (
    <div className="rounded-lg border border-sand-200 bg-[#FFFDFA] p-4">
      <div className={MICRO_LABEL}>{label}</div>
      <div className={cn("mt-2 text-[20px] font-semibold", tone === "ok" ? "text-[#2F7A57]" : "text-cocoa-900")}>{value}</div>
    </div>
  );
}

function AlphaMini({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border border-sand-200 bg-[#FFFDFA] p-3">
      <div className={MICRO_LABEL}>{label}</div>
      <div className="mt-1 text-[15px] font-semibold text-cocoa-900">{value}</div>
    </div>
  );
}

function percent(value?: number) {
  if (value == null) return "0%";
  return `${Math.round(value * 100)}%`;
}

function statusBadge(tone: "ok" | "warning" | "neutral") {
  if (tone === "ok") return "rounded-full bg-[#EDF3EA] px-2.5 py-1 font-semibold text-[#2F7A57]";
  if (tone === "warning") return "rounded-full bg-[#FAEFDA] px-2.5 py-1 font-semibold text-[#96682A]";
  return "rounded-full bg-[#F4EDE4] px-2.5 py-1 font-semibold text-[#8A7A6A]";
}
