import type { GenerationJob } from "@/entities/generation-job/model";
import type { Trip } from "@/entities/trip/model";
import type { TripPoll } from "@/types/trip-decisions";

export type AvailabilityDateRange = {
  startDate: string;
  endDate: string;
};

export type TripAvailabilityResponseInfo = {
  userId: string;
  displayName: string;
  availableRanges: AvailabilityDateRange[];
  unavailableRanges: AvailabilityDateRange[];
  preferredRanges: AvailabilityDateRange[];
  minTripDays?: number | null;
  maxTripDays?: number | null;
  timezone?: string;
  notes?: string;
  submitted: boolean;
  updatedAt?: string | null;
};

export type TripAvailabilityUserSummary = {
  userId: string;
  displayName: string;
};

export type TripAvailabilitySummary = {
  totalCollaborators: number;
  submittedCount: number;
  missingCount: number;
  missingUsers: TripAvailabilityUserSummary[];
};

export type TripAvailabilityList = {
  tripId: string;
  responses: TripAvailabilityResponseInfo[];
  summary: TripAvailabilitySummary;
};

export type UpsertTripAvailabilityInput = {
  availableRanges: AvailabilityDateRange[];
  unavailableRanges?: AvailabilityDateRange[];
  preferredRanges?: AvailabilityDateRange[];
  minTripDays?: number | null;
  maxTripDays?: number | null;
  timezone?: string;
  notes?: string;
};

export type DateOptionUserSummary = {
  userId: string;
  displayName: string;
};

export type DateOptionConflict = {
  userId: string;
  displayName: string;
  reason: string;
};

export type TripDateOption = {
  id: string;
  startDate: string;
  endDate: string;
  durationDays: number;
  score: number;
  availableUserCount: number;
  totalUserCount: number;
  preferredUserCount: number;
  conflictUserCount: number;
  missingResponseUserCount: number;
  availableUsers: DateOptionUserSummary[];
  conflicts: DateOptionConflict[];
  missingResponses: DateOptionUserSummary[];
  pros: string[];
  cons: string[];
  warnings: string[];
};

export type DateOptionsInput = {
  minDays?: number | null;
  maxDays?: number | null;
  searchStartDate?: string;
  searchEndDate?: string;
  preferWeekends?: boolean | null;
  limit?: number;
};

export type DateOptionsSummary = {
  responseCount: number;
  totalCollaborators: number;
  recommendedOptionId?: string;
  missingResponseCount: number;
};

export type DateOptionsResult = {
  options: TripDateOption[];
  summary: DateOptionsSummary;
};

export type ApplyDateOptionInput = {
  expectedItineraryRevision?: number;
  regenerateItinerary?: boolean;
};

export type ApplyDateOptionResponse = {
  trip: Trip;
  appliedOption: TripDateOption;
  itineraryStale: boolean;
  routeShifted: boolean;
  warnings: string[];
  generationJob?: GenerationJob | null;
};

export type CreateDateOptionsPollInput = {
  title?: string;
  optionIds: string[];
};

export type RequestTripAvailabilityInput = {
  message?: string;
};

export type CreateDateOptionsPollResult = TripPoll;
