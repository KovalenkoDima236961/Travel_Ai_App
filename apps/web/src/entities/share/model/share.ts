import type { Itinerary, TripStatus } from "@/entities/trip/model";

export type TripShareInfo = {
  shareToken?: string | null;
  shareUrl?: string | null;
  enabled: boolean;
  createdAt?: string | null;
  updatedAt?: string | null;
  disabledAt?: string | null;
  expiresAt?: string | null;
  expired?: boolean;
  passwordRequired: boolean;
};

export type UpdateTripShareRequest = {
  expiresAt?: string | null;
  clearExpiration?: boolean;
  password?: string;
  clearPassword?: boolean;
};

export type PublicShareStatus = {
  available: boolean;
  passwordRequired: boolean;
  expired?: boolean;
};

export type PublicShareUnlockResponse = {
  accessToken: string;
  expiresAt: string;
};

// PublicTrip intentionally omits the private trip budget (amount and currency).
// Item-level estimated costs remain inside the shared itinerary.
export type PublicTrip = {
  destination: string;
  startDate?: string | null;
  days: number;
  travelers?: number | null;
  interests?: string[];
  pace?: string | null;
  status: TripStatus;
  itinerary?: Itinerary | null;
  createdAt?: string | null;
  updatedAt?: string | null;
  sharedAt?: string | null;
};
