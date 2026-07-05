export type TripTravelerRole = "organizer" | "traveler";
export type TripTravelerStatus = "active" | "removed";

export type TripTraveler = {
  id: string;
  tripId: string;
  name: string;
  email?: string | null;
  linkedUserId?: string | null;
  role: TripTravelerRole;
  status: TripTravelerStatus;
  createdAt: string;
  updatedAt: string;
};

export type CreateTripTravelerInput = {
  name: string;
  email?: string | null;
  linkedUserId?: string | null;
  role: TripTravelerRole;
};

export type UpdateTripTravelerInput = {
  name?: string;
  email?: string | null;
  role?: TripTravelerRole;
};

export type CostSplitType = "all_equal" | "selected_equal" | "custom_percentages";

export type CostSplitRule = {
  type: CostSplitType;
  travelerIds?: string[];
  percentages?: Record<string, number>;
};

export type CostSplittingSummary = {
  tripId: string;
  currency: string;
  generatedAt: string;
  summary: CostSplittingTotals;
  travelers: TravelerCostAllocation[];
  unassignedCosts: UnassignedCost[];
  byCategory: CostSplitCategoryTotal[];
  byDay: CostSplitDayTotal[];
  warnings: string[];
  exchangeRateInfo?: CostSplitExchangeRateInfo | null;
};

export type CostSplitExchangeRateInfo = {
  provider?: string | null;
  asOf?: string | null;
  fallbackUsed?: boolean;
};

export type CostSplittingTotals = {
  travelerCount: number;
  estimatedTotal: number;
  allocatedTotal: number;
  unassignedTotal: number;
  missingEstimateCount: number;
  defaultSplitCount: number;
  invalidSplitCount: number;
  convertedItemCount: number;
  unconvertedItemCount: number;
};

export type TravelerCostAllocation = {
  travelerId: string;
  name: string;
  email?: string | null;
  linkedUserId?: string | null;
  role: TripTravelerRole;
  allocatedTotal: number;
  percentageOfTotal: number;
  byCategory: CostSplitCategoryTotal[];
  byDay: CostSplitDayTotal[];
  items: TravelerAllocatedItem[];
};

export type TravelerAllocatedItem = {
  type: "itinerary_item" | "accommodation" | string;
  dayNumber?: number | null;
  itemIndex?: number | null;
  name: string;
  category: string;
  allocatedAmount: number;
  originalCostAmount: number;
  originalCostCurrency: string;
  splitType: CostSplitType | string;
  ruleSource: "explicit" | "default" | string;
};

export type UnassignedCost = {
  type: "itinerary_item" | "accommodation" | string;
  dayNumber?: number | null;
  itemIndex?: number | null;
  name: string;
  amount: number;
  currency: string;
  reason: string;
};

export type CostSplitCategoryTotal = {
  category: string;
  amount: number;
};

export type CostSplitDayTotal = {
  dayNumber: number;
  amount: number;
};

export type TripTravelersResponse = {
  travelers: TripTraveler[];
};
