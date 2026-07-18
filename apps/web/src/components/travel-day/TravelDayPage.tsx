"use client";

import dynamic from "next/dynamic";
import { useState } from "react";
import { useAuth } from "@/components/auth/AuthProvider";
import { MobileExpenseQuickAdd } from "./MobileExpenseQuickAdd";
import { AccommodationQuickCard } from "./AccommodationQuickCard";
import { NowNextCard } from "./NowNextCard";
import { OfflineTravelStatus } from "./OfflineTravelStatus";
import { TodayTimeline } from "./TodayTimeline";
import { TransportQuickCard } from "./TransportQuickCard";
import { TravelChecklistMiniPanel } from "./TravelChecklistMiniPanel";
import { TravelDayEmptyState } from "./TravelDayEmptyState";
import { TravelDayErrorState } from "./TravelDayErrorState";
import { TravelDayHeader } from "./TravelDayHeader";
import { TravelDaySkeleton } from "./TravelDaySkeleton";
import { TravelDayCopilotShortcut } from "./TravelDayCopilotShortcut";
import { TravelQuickActions } from "./TravelQuickActions";
import { TravelRemindersMiniPanel } from "./TravelRemindersMiniPanel";
import { TravelVerificationBadge } from "./TravelVerificationBadge";
import { TravelWarningsCard } from "./TravelWarningsCard";
import { TravelWeatherStrip } from "./TravelWeatherStrip";
import { useNetworkStatus } from "@/hooks/useNetworkStatus";
import { useTodayDate } from "@/hooks/useTodayDate";
import { useTravelDay } from "@/hooks/useTravelDay";
import { useUpdateTravelItemStatus } from "@/hooks/useUpdateTravelItemStatus";
import type { TravelDayTimelineItem } from "@/types/travel-day";

const UploadReceiptDialog = dynamic(
  () => import("@/components/receipts").then((module) => module.UploadReceiptDialog),
  { ssr: false }
);

export function TravelDayPage({ tripId }: { tripId: string }) {
  const { user } = useAuth();
  const date = useTodayDate();
  const network = useNetworkStatus();
  const [receiptOpen, setReceiptOpen] = useState(false);
  const [copilotOpen, setCopilotOpen] = useState(false);
  const query = useTravelDay({ tripId, date, userId: user?.id });
  const summary = query.data?.summary;
  const statusMutation = useUpdateTravelItemStatus({ tripId, date, userId: user?.id, offline: !network.online });

  function updateStatus(item: TravelDayTimelineItem, status: "done" | "skipped" | "delayed") {
    statusMutation.mutate({ dayNumber: item.dayNumber, itemIndex: item.itemIndex, status });
  }
  function openMap(item: TravelDayTimelineItem) {
    const place = item.place;
    const queryText = place?.latitude != null && place.longitude != null
      ? `${place.latitude},${place.longitude}`
      : item.locationName || place?.address || item.title;
    window.open(`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(queryText)}`, "_blank", "noopener,noreferrer");
  }

  if (query.isLoading) return <TravelDaySkeleton />;
  if (!summary) return <TravelDayErrorState onRetry={() => void query.refetch()} />;
  if (!summary.timeline.length && summary.mode !== "active") return <TravelDayEmptyState mode={summary.mode} tripId={tripId} />;

  return <main className="mx-auto max-w-5xl space-y-4 px-4 pb-28 pt-5 sm:px-6 lg:grid lg:grid-cols-[minmax(0,1fr)_320px] lg:items-start lg:gap-5 lg:space-y-0"><div className="min-w-0 space-y-4"><TravelDayHeader summary={summary}/><OfflineTravelStatus cachedAt={query.data?.cachedAt} offlineCopy={Boolean(query.data?.offlineCopy)} summary={summary}/><NowNextCard busy={statusMutation.isPending} canUpdate={summary.permissions.canUpdateTravelStatus} nowNext={summary.nowNext} onMap={openMap} onStatus={updateStatus}/><TravelWarningsCard warnings={[...summary.verification.topWarnings, ...summary.weather.warnings]}/><TodayTimeline busy={statusMutation.isPending} canUpdate={summary.permissions.canUpdateTravelStatus} items={summary.timeline} onMap={openMap} onStatus={updateStatus}/></div><aside className="mt-4 space-y-4 lg:mt-0"><div className="flex items-center justify-between"><TravelVerificationBadge verification={summary.verification}/><span className="text-xs text-cocoa-500">{summary.today.primaryLocation}</span></div><TravelWeatherStrip weather={summary.weather}/><TransportQuickCard route={summary.route}/><AccommodationQuickCard accommodation={summary.accommodation}/><TravelChecklistMiniPanel checklist={summary.checklist} online={network.online} tripId={tripId}/><TravelRemindersMiniPanel online={network.online} reminders={summary.reminders} tripId={tripId}/><MobileExpenseQuickAdd currency={summary.expenses.quickAddDefaults.currency} online={network.online} tripId={tripId} userId={user?.id}/></aside><TravelQuickActions onCopilot={() => setCopilotOpen(true)} onReceipt={() => setReceiptOpen(true)} tripId={tripId}/>{receiptOpen && user ? <div className="fixed inset-0 z-40 overflow-y-auto bg-cocoa-900/30 p-4"><div className="mx-auto mt-10 max-w-2xl"><UploadReceiptDialog currency={summary.expenses.quickAddDefaults.currency} onClose={() => setReceiptOpen(false)} users={[{ id: user.id, name: user.email }]} tripId={tripId}/></div></div> : null}{copilotOpen ? <TravelDayCopilotShortcut summary={summary}/> : null}</main>;
}
