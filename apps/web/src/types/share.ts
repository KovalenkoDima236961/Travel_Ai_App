import type { Itinerary, TripStatus } from "@/types/trip";

export type TripShareInfo = {
  shareToken?: string | null;
  shareUrl?: string | null;
  enabled: boolean;
  createdAt?: string | null;
  disabledAt?: string | null;
};

export type PublicTrip = {
  destination: string;
  startDate?: string | null;
  days: number;
  budgetAmount?: number | null;
  budgetCurrency?: string | null;
  travelers?: number | null;
  interests?: string[];
  pace?: string | null;
  status: TripStatus;
  itinerary?: Itinerary | null;
  createdAt?: string | null;
  updatedAt?: string | null;
  sharedAt?: string | null;
};
