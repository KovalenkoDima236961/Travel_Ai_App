import type { TripPresenceSnapshot, TripPresenceUser } from "@/types/presence";

export function getPresenceDisplayName(
  user: TripPresenceUser,
  currentUserId?: string | null
) {
  if (currentUserId && user.userId === currentUserId) {
    return "You";
  }
  const displayName = user.displayName?.trim();
  return displayName || "Collaborator";
}

export function getOtherEditingUsers(
  snapshot: TripPresenceSnapshot | null,
  currentUserId?: string | null
) {
  return (snapshot?.users ?? []).filter(
    (user) => user.state === "editing" && (!currentUserId || user.userId !== currentUserId)
  );
}

export function getPresenceEditingWarning(
  snapshot: TripPresenceSnapshot | null,
  currentUserId?: string | null
) {
  const editors = getOtherEditingUsers(snapshot, currentUserId);
  if (editors.length === 0) {
    return null;
  }
  if (editors.length === 1) {
    const name = getPresenceDisplayName(editors[0], currentUserId);
    return `${name} is currently editing this itinerary. Be careful before saving changes.`;
  }
  return `${editors.length} collaborators are currently editing this itinerary.`;
}
