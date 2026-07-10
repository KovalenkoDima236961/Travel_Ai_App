export type TripPollType =
  | "single_choice"
  | "multiple_choice"
  | "rating"
  | "yes_no"
  | "date_choice";

export type TripPollStatus = "open" | "closed" | "archived";

export type ItineraryReaction = "must_have" | "want_to_do" | "neutral" | "skip";

export type DiscoverySuggestionVoteValue =
  | "like"
  | "dislike"
  | "favorite"
  | "not_interested";

export type TripPollOption = {
  id: string;
  optionKey: string;
  label: string;
  description?: string;
  sortOrder: number;
  metadata?: Record<string, unknown>;
  createdAt: string;
};

export type TripPollVote = {
  id: string;
  optionId?: string;
  voteValue?: string;
  ratingValue?: number;
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
};

export type PollOptionResult = {
  optionId: string;
  optionKey: string;
  label: string;
  voteCount: number;
  percentage: number;
  averageRating?: number;
};

export type TripPollResults = {
  totalVoters: number;
  totalVotes: number;
  options: PollOptionResult[];
  winningOptionIds: string[];
};

export type TripPoll = {
  id: string;
  tripId: string;
  title: string;
  description?: string;
  pollType: TripPollType;
  status: TripPollStatus;
  allowMultipleVotes: boolean;
  options: TripPollOption[];
  results: TripPollResults;
  userVotes: TripPollVote[];
  createdByUserId: string;
  createdAt: string;
  updatedAt: string;
  closesAt?: string;
  closedAt?: string;
  closedByUserId?: string;
  canManage: boolean;
  canVote: boolean;
  metadata?: Record<string, unknown>;
};

export type CreateTripPollInput = {
  title: string;
  description?: string;
  pollType: TripPollType;
  allowMultipleVotes?: boolean;
  closesAt?: string;
  metadata?: Record<string, unknown>;
  options: Array<{
    optionKey?: string;
    label: string;
    description?: string;
    metadata?: Record<string, unknown>;
  }>;
};

export type VoteTripPollInput = {
  optionIds?: string[];
  voteValue?: string | null;
  ratingValue?: number | null;
  metadata?: Record<string, unknown>;
};

export type ItineraryItemReactionSummary = {
  dayNumber: number;
  itemIndex: number;
  itemId?: string;
  itemName?: string;
  counts: Record<ItineraryReaction, number>;
  currentUserReaction?: ItineraryReaction;
  score: number;
};

export type SetItineraryItemReactionInput = {
  dayNumber: number;
  itemIndex: number;
  itemId?: string;
  reaction: ItineraryReaction;
  metadata?: Record<string, unknown>;
};

export type GroupPreferenceScore = {
  key: string;
  label: string;
  score: number;
  votes: number;
};

export type GroupPreferenceItineraryItem = {
  dayNumber: number;
  itemIndex: number;
  itemId?: string;
  name: string;
  count: number;
  score: number;
};

export type GroupPreferencesSummary = {
  tripId: string;
  generatedAt: string;
  summary: {
    collaboratorCount: number;
    pollCount: number;
    openPollCount: number;
    reactionCount: number;
    mustHaveItemCount: number;
    skipItemCount: number;
    openDecisionCount: number;
  };
  topPollChoices: Array<{
    pollId: string;
    title: string;
    pollType: TripPollType;
    winningOptions: Array<{
      optionId: string;
      optionKey: string;
      label: string;
      voteCount: number;
      percentage: number;
    }>;
  }>;
  itineraryPreferences: {
    mustHaveItems: GroupPreferenceItineraryItem[];
    mostSkippedItems: GroupPreferenceItineraryItem[];
    controversial: GroupPreferenceItineraryItem[];
  };
  transportPreferences: GroupPreferenceScore[];
  destinationPreferences: GroupPreferenceScore[];
  datePreferences: GroupPreferenceScore[];
  aiConstraintSummary: string;
};

export type DiscoverySuggestionVoteSummary = {
  suggestionId: string;
  counts: Record<DiscoverySuggestionVoteValue, number>;
  score: number;
  currentUserVote?: DiscoverySuggestionVoteValue;
};

export type DiscoverySuggestionVotesResponse = {
  sessionId: string;
  items: DiscoverySuggestionVoteSummary[];
};
