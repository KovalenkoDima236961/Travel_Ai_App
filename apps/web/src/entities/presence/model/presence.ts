export type PresenceState = "viewing" | "editing";

export type TripPresenceUser = {
  userId: string;
  displayName?: string | null;
  role: "owner" | "editor" | "viewer";
  state: PresenceState;
  connectedAt: string;
  lastSeenAt: string;
};

export type TripPresenceSnapshot = {
  tripId: string;
  users: TripPresenceUser[];
};
