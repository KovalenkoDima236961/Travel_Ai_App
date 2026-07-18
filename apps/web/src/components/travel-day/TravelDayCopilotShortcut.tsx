"use client";

import { TripCopilot } from "@/components/copilot";
import type { TravelDaySummary } from "@/types/travel-day";
export function TravelDayCopilotShortcut({ summary }: { summary: TravelDaySummary }) { return <TripCopilot clientContext={{ currentTab: "travel_day", date: summary.date, dayNumber: summary.dayNumber, currentItemId: summary.nowNext.currentItem?.itemId, nextItemId: summary.nowNext.nextItem?.itemId }} currentPath={`/trips/${summary.tripId}/today`} currentTab="travel_day" openOnMount tripId={summary.tripId} />; }
