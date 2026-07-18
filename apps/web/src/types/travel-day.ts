import type { TripAccommodation } from "@/entities/accommodation/model";
import type { EstimatedCost } from "@/entities/budget/model";
import type { TripChecklistItem } from "@/entities/checklist/model";
import type { TripExpense } from "@/entities/expense/model";
import type { Place } from "@/entities/place/model";
import type { TripRoute } from "@/entities/route/model";
import type { SelectedTransportOption } from "@/types/transport";
import type { TripReminder } from "@/entities/trip-reminder/model";
import type { ItineraryTravelStatus, TravelStatus } from "@/entities/trip/model";
import type { VerificationDetail } from "@/types/verification";

export type TravelDayAction = { type: string; label: string; href?: string };

export type TravelDayTimelineItem = {
  dayNumber: number;
  itemIndex: number;
  itemId?: string;
  startTime?: string;
  endTime?: string;
  title: string;
  type: string;
  description?: string;
  locationName?: string;
  place?: Place | null;
  selectedTransport?: SelectedTransportOption | null;
  estimatedCost?: EstimatedCost | null;
  travelStatus: ItineraryTravelStatus;
  verification?: VerificationDetail[];
  actions: TravelDayAction[];
};

export type TravelDayNowNext = {
  currentItem?: TravelDayTimelineItem | null;
  nextItem?: TravelDayTimelineItem | null;
  afterNextItems: TravelDayTimelineItem[];
};

export type TravelWarning = VerificationDetail;
export type TravelWeatherImpact = VerificationDetail;

export type TravelDayPermissionSummary = {
  canUpdateTravelStatus: boolean;
  canAddExpense: boolean;
  canUploadReceipt: boolean;
  canEditTrip: boolean;
};

export type TravelDaySummary = {
  tripId: string;
  date: string;
  dayNumber: number;
  mode: "active" | "pre_trip" | "post_trip" | string;
  timezone: string;
  trip: { title: string; destination: string; startDate?: string; endDate?: string; tripType: string };
  today: { title: string; primaryLocation?: string; summary?: string };
  nowNext: TravelDayNowNext;
  timeline: TravelDayTimelineItem[];
  route: { todayLegs: NonNullable<TripRoute["legs"]>; selectedTransportSummary: SelectedTransportOption[] };
  weather: { summary: string; warnings: TravelWarning[] };
  verification: { score: number; level?: string; topWarnings: TravelWarning[]; unavailable?: boolean };
  checklist: { dueToday: TripChecklistItem[]; overdue: TripChecklistItem[]; progress: { completed: number; total: number } };
  reminders: { dueToday: TripReminder[]; overdue: TripReminder[] };
  accommodation?: TripAccommodation | null;
  expenses: { todayTotal: { amount: number; currency: string }; quickAddDefaults: { currency: string } };
  offline: { cacheRecommended: boolean; lastCachedAt?: string };
  permissions: TravelDayPermissionSummary;
  sectionErrors: Array<{ section: string; code: string }>;
  generatedAt: string;
  itineraryRevision: number;
};

export type UpdateTravelItemStatusInput = {
  status: TravelStatus;
  note?: string;
  expectedItineraryRevision: number;
};

export type UpdateTravelItemStatusResponse = {
  status: TravelStatus;
  updatedAt: string;
  updatedByUserId: string;
  note?: string;
  itineraryRevision: number;
};

// Kept here for quick-add consumers that only need a narrow expense shape.
export type TravelDayExpenseDraft = Pick<TripExpense, "title" | "category">;
