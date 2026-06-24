export type ItineraryComment = {
  id: string;
  tripId: string;
  dayNumber: number;
  itemIndex: number;
  authorUserId: string;
  authorDisplayName?: string | null;
  authorEmail?: string | null;
  body: string;
  createdAt: string;
  updatedAt: string;
  canEdit?: boolean;
  canDelete?: boolean;
  isAuthor?: boolean;
};

export type CommentCount = {
  dayNumber: number;
  itemIndex: number;
  count: number;
};

export type CreateCommentRequest = {
  dayNumber: number;
  itemIndex: number;
  body: string;
};

export type UpdateCommentRequest = {
  body: string;
};
