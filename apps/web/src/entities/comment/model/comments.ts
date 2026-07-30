export type ItineraryComment = {
  id: string;
  tripId: string;
  dayNumber: number;
  itemIndex: number;
  targetType: CommentTargetType;
  targetId?: string | null;
  parentCommentId?: string | null;
  authorUserId: string;
  authorDisplayName?: string | null;
  authorEmail?: string | null;
  body: string;
  mentionUserIds?: string[];
  attachments?: string[];
  resolvedAt?: string | null;
  resolvedByUserId?: string | null;
  editedAt?: string | null;
  createdAt: string;
  updatedAt: string;
  canEdit?: boolean;
  canDelete?: boolean;
  isAuthor?: boolean;
};

export type CommentTargetType =
  | "trip"
  | "day"
  | "itinerary_item"
  | "budget_item"
  | "route"
  | "attachment";

export type CommentCount = {
  dayNumber: number;
  itemIndex: number;
  count: number;
};

export type CreateCommentRequest = {
  dayNumber?: number;
  itemIndex?: number;
  targetType?: CommentTargetType;
  targetId?: string;
  parentCommentId?: string;
  mentionUserIds?: string[];
  body: string;
};

export type UpdateCommentRequest = {
  body: string;
};
