import type { EstimatedCost } from "@/entities/budget/model";
import type { Place } from "@/entities/place/model";

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
