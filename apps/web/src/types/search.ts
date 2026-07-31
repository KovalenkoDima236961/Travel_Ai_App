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
  matchedFields?: string[];
  sourceService?: string;
  metadata?: Record<string, unknown>;
};

export type SearchResultGroup = {
  title: string;
  items: SearchResult[];
};

export type SearchResponse = {
  query: string;
  data?: SearchResult[];
  items: SearchResult[];
  groups: SearchResultGroup[];
  hasMore: boolean;
  pagination?: {
    nextCursor: string | null;
    hasMore: boolean;
    limit: number;
  };
  queryMeta?: {
    normalized: string;
    scope: SearchScope;
    types?: SearchResultType[];
    includeArchived: boolean;
  };
};

export type SearchParams = {
  q: string;
  scope?: SearchScope;
  types?: SearchResultType[];
  tripId?: string | null;
  workspaceId?: string | null;
  limit?: number;
  includeArchived?: boolean;
  includeCommands?: boolean;
};
