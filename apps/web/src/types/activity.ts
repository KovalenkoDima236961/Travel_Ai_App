export type TripActivityEventType =
  | "trip_created"
  | "itinerary_generated"
  | "itinerary_updated"
  | "day_regenerated"
  | "item_regenerated"
  | "version_restored"
  | "generation_job_failed"
  | "comment_created"
  | "comment_updated"
  | "comment_deleted"
  | "collaborator_invited"
  | "collaborator_accepted"
  | "collaborator_declined"
  | "collaborator_role_changed"
  | "collaborator_removed"
  | "share_created"
  | "share_updated"
  | "share_disabled"
  | "accommodation_added"
  | "accommodation_updated"
  | "accommodation_removed";

export type TripActivityEntityType =
  | "trip"
  | "itinerary"
  | "itinerary_day"
  | "itinerary_item"
  | "itinerary_version"
  | "comment"
  | "collaborator"
  | "share"
  | "accommodation";

export type TripActivityEvent = {
  id: string;
  tripId: string;
  actorUserId?: string | null;
  // The backend may emit event types beyond this version's known set; treat the
  // type as open so an unrecognised value never breaks rendering.
  eventType: TripActivityEventType | (string & {});
  entityType?: TripActivityEntityType | null;
  entityId?: string | null;
  metadata: Record<string, unknown>;
  createdAt: string;
};

export type TripActivityResponse = {
  items: TripActivityEvent[];
  nextCursor?: string | null;
};
