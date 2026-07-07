export type EditLockScope = "itinerary";

export type EditLockView = {
  locked: boolean;
  scope: EditLockScope;
  tripId: string;
  lockedByUserId?: string | null;
  lockedByDisplayName?: string | null;
  lockedByRole?: "owner" | "editor" | "viewer" | null;
  lockedByCurrentUser?: boolean;
  createdAt?: string | null;
  expiresAt?: string | null;
  ttlSeconds?: number | null;
  disabled?: boolean;
};

export type AcquireEditLockResponse = {
  acquired: boolean;
  renewed?: boolean;
  disabled?: boolean;
  reason?: string | null;
  lock?: EditLockView | null;
};

export type ReleaseEditLockResponse = {
  released: boolean;
};
