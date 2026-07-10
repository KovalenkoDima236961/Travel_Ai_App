import type { TransportMode, TripStyle } from "@/entities/route/model";

export const transportModeOptions: Array<{ value: TransportMode; label: string }> = [
  { value: "walk", label: "Walking" },
  { value: "car", label: "Car" },
  { value: "rental_car", label: "Rental car" },
  { value: "train", label: "Train" },
  { value: "bus", label: "Bus" },
  { value: "flight", label: "Flight" },
  { value: "ferry", label: "Ferry/boat" },
  { value: "bike", label: "Bike" },
  { value: "hiking", label: "Hiking" },
  { value: "public_transport", label: "Public transport" },
  { value: "other", label: "Other" }
];

export const tripStyleOptions: Array<{ value: TripStyle; label: string }> = [
  { value: "road_trip", label: "Road trip" },
  { value: "train_trip", label: "Train trip" },
  { value: "backpacking", label: "Backpacking" },
  { value: "camping", label: "Camping" },
  { value: "hiking", label: "Hiking" },
  { value: "island_hopping", label: "Island hopping" },
  { value: "nature", label: "Nature" },
  { value: "beach", label: "Beach" },
  { value: "food", label: "Food" },
  { value: "culture", label: "Culture" },
  { value: "adventure", label: "Adventure" },
  { value: "family", label: "Family" },
  { value: "romantic", label: "Romantic" },
  { value: "low_budget", label: "Low budget" },
  { value: "hidden_gem", label: "Hidden gem" }
];

export function transportModeLabel(mode: string | null | undefined) {
  return transportModeOptions.find((option) => option.value === mode)?.label ?? "Other";
}

export function tripStyleLabel(style: string | null | undefined) {
  return tripStyleOptions.find((option) => option.value === style)?.label ?? style ?? "";
}
