import type {
  AvailabilityDateRange,
  DateOptionsResult,
  TripAvailabilityResponseInfo,
  UpsertTripAvailabilityInput
} from "@/types/trip-availability";

export type CalendarFreeBusyBlock = {
  start: string;
  end: string;
  allDay: boolean;
  source: "google_calendar";
};

export type CalendarFreeBusySummary = {
  startDate: string;
  endDate: string;
  timezone: string;
  busyBlockCount: number;
  busyDays: number;
  fullyBusyDays: number;
  partiallyBusyDays: number;
  calendarCount: number;
};

export type CalendarImportConversionSettings = {
  fullyBusyThresholdHours: number;
  markFullyBusyDaysUnavailable: boolean;
  markPartiallyBusyDaysUnavailable: boolean;
  includeWeekendsAsPreferredIfFree: boolean;
};

export type CalendarImportMode = "merge" | "overwrite_all_my_availability";

export type CalendarImportPreviewRequest = {
  startDate: string;
  endDate: string;
  timezone: string;
  calendarProvider: "google";
  calendarIds?: string[];
  conversion: CalendarImportConversionSettings;
};

export type CalendarImportApplyRequest = CalendarImportPreviewRequest & {
  mode: CalendarImportMode;
  availabilitySettings: UpsertTripAvailabilityInput;
};

export type CalendarBusyDaySummary = {
  date: string;
  status: "fully_busy" | "partially_busy" | "free";
  busyHours: number;
  busyBlockCount: number;
};

export type CalendarImportSuggestedRange = AvailabilityDateRange & {
  reason: "calendar_fully_busy" | "calendar_free_window" | string;
};

export type CalendarImportPreview = {
  source: "google_calendar";
  range: {
    startDate: string;
    endDate: string;
    timezone: string;
  };
  busyBlocksSummary: {
    busyBlockCount: number;
    busyDays: number;
    fullyBusyDays: number;
    partiallyBusyDays: number;
  };
  suggestedUnavailableRanges: CalendarImportSuggestedRange[];
  suggestedPreferredRanges: CalendarImportSuggestedRange[];
  daySummaries: CalendarBusyDaySummary[];
  warnings: string[];
};

export type CalendarImportPreviewResponse = {
  preview: CalendarImportPreview;
};

export type CalendarImportApplyResponse = {
  availability: TripAvailabilityResponseInfo;
  dateOptions: DateOptionsResult;
};
