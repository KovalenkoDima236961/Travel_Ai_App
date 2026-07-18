export type TripLifecycle = "draft" | "planning" | "ready" | "active" | "completed" | "archived";

export type LibraryMoney = { amount: number; currency: string };

export type TripLibraryItem = {
  trip: {
    id: string;
    destination: string;
    startDate?: string | null;
    days: number;
    tripType: string;
    workspaceId?: string | null;
    archivedAt?: string | null;
    archivedByUserId?: string | null;
    updatedAt: string;
  };
  lifecycle: TripLifecycle;
  recap: { hasRecap: boolean; status?: string; href?: string; createdAt?: string | null };
  template: { hasTemplate: boolean; templateId?: string | null };
  budget: {
    plannedTotal?: LibraryMoney | null;
    actualTotal?: LibraryMoney | null;
    variance?: LibraryMoney | null;
    mixedCurrencies?: boolean;
  };
  completion: { plannedItemCount: number; doneItemCount: number; completionRate: number };
  route: { transportModes: string[]; stopCount: number };
  actions: string[];
};

export type TripLibrarySort =
  | "recently_updated"
  | "trip_date_desc"
  | "trip_date_asc"
  | "destination"
  | "budget_desc"
  | "budget_asc"
  | "completion_rate_desc"
  | "recap_created_desc";

export type TripLibraryFilters = {
  q?: string;
  lifecycle?: TripLifecycle | "all" | "active,planning,ready,draft";
  workspaceId?: string;
  year?: number;
  destination?: string;
  country?: string;
  tripType?: string;
  travelStyle?: string;
  transportMode?: string;
  budgetMin?: number;
  budgetMax?: number;
  currency?: string;
  hasRecap?: boolean;
  hasTemplate?: boolean;
  hasExpenses?: boolean;
  archived?: boolean;
  sort?: TripLibrarySort;
  limit?: number;
  cursor?: string;
};

export type TripLibraryResponse = {
  items: TripLibraryItem[];
  nextCursor?: string;
  filters: { availableYears: number[]; availableDestinations: string[] };
  summary: { total: number; completed: number; archived: number; withRecaps: number; withTemplates: number };
};

export type TripLibraryInsights = {
  summary: { tripCount: number; completedTripCount: number; archivedTripCount: number; totalTravelDays: number; countriesVisitedCount: number };
  topDestinations: Array<{ label: string; count: number }>;
  topCountries: Array<{ label: string; count: number }>;
  budget: { averageTripBudget?: LibraryMoney | null; averageActualSpend?: LibraryMoney | null; underBudgetTripCount: number; overBudgetTripCount: number; mixedCurrencies?: boolean };
  transportModes: Array<{ label: string; count: number }>;
  travelStyles: Array<{ label: string; count: number }>;
  recaps: { tripRecapCount: number; commonLessons: string[] };
  templates: { templatesCreatedFromTrips: number };
  checklists: { commonlyMissedItems: Array<{ label: string; count: number }> };
};

export type ArchiveTripInput = { reason?: string };
export type ArchiveTripResponse = { tripId: string; archivedAt: string | null; lifecycle: TripLifecycle };
