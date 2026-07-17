export type SearchResultType =
  | "trip"
  | "workspace"
  | "template"
  | "itinerary_item"
  | "route_stop"
  | "route_leg"
  | "transport_option"
  | "expense"
  | "receipt"
  | "checklist_item"
  | "reminder"
  | "poll"
  | "collaborator"
  | "notification"
  | "setting"
  | "command"
  | "ops_page";

export type SearchScope = "all" | "trips" | "current_trip" | "workspace" | "ops";

export type SearchResult = {
  id: string;
  type: SearchResultType;
  title: string;
  description?: string;
  context?: string;
  workspaceName?: string;
  href: string;
  icon: string;
  category: string;
  score: number;
  metadata?: Record<string, unknown>;
};

export type SearchResultGroup = {
  title: string;
  items: SearchResult[];
};

export type SearchResponse = {
  query: string;
  items: SearchResult[];
  groups: SearchResultGroup[];
  hasMore: boolean;
};

export type SearchParams = {
  q: string;
  scope?: SearchScope;
  tripId?: string | null;
  workspaceId?: string | null;
  limit?: number;
  includeCommands?: boolean;
};
