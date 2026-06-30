import type { EstimatedCost } from "@/types/budget";
import type { Place } from "@/types/place";

export type AccommodationType =
  | "hotel"
  | "hostel"
  | "apartment"
  | "guesthouse"
  | "home"
  | "other";

export type TripAccommodation = {
  name: string;
  type: AccommodationType;
  address?: string | null;
  place?: Place | null;
  checkInDate?: string | null;
  checkOutDate?: string | null;
  estimatedCost?: EstimatedCost | null;
  notes?: string | null;
};

export const ACCOMMODATION_TYPES: AccommodationType[] = [
  "hotel",
  "hostel",
  "apartment",
  "guesthouse",
  "home",
  "other"
];
