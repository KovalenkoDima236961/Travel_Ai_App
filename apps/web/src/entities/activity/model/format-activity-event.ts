import type { TripActivityEvent } from "@/entities/activity/model";

export type FormattedActivityEvent = {
  actorLabel: string;
  title: string;
  description?: string;
  dayNumber?: number;
  itemIndex?: number;
  createdAt: string;
};

/**
 * Turns a raw activity event into a short, human-readable line for the feed.
 *
 * The actor is rendered relative to the current viewer: "You" for the viewer's
 * own actions, "System" for actor-less events, and "Collaborator" otherwise
 * (display names are not resolved in v1). The formatter is defensive: metadata
 * fields may be missing, and an unknown event type falls back to a generic
 * "Activity recorded" line rather than throwing.
 */
export function formatActivityEvent(
  event: TripActivityEvent,
  currentUserId?: string | null
): FormattedActivityEvent {
  const actorLabel = resolveActorLabel(event.actorUserId, currentUserId);
  const dayNumber = asNumber(event.metadata?.dayNumber);
  const itemIndex = asNumber(event.metadata?.itemIndex);

  const base: FormattedActivityEvent = {
    actorLabel,
    title: "Activity recorded",
    createdAt: event.createdAt
  };
  if (dayNumber != null) {
    base.dayNumber = dayNumber;
  }
  if (itemIndex != null) {
    base.itemIndex = itemIndex;
  }

  return { ...base, ...titleFor(event, actorLabel) };
}

function titleFor(
  event: TripActivityEvent,
  actor: string
): { title: string; description?: string } {
  const metadata = event.metadata ?? {};

  switch (event.eventType) {
    case "trip_created":
      return { title: `${actor} created the trip` };
    case "trip_budget_updated":
      return { title: `${actor} updated the trip budget` };
    case "itinerary_generated":
      return { title: `${actor} generated the itinerary` };
    case "itinerary_updated":
      return { title: `${actor} updated the itinerary` };
    case "day_regenerated":
      return { title: `${actor} regenerated ${dayLabel(metadata)}` };
    case "item_regenerated": {
      const name = asString(metadata.itemName);
      const target = name ? `${dayLabel(metadata)} item: ${name}` : `${dayLabel(metadata)}`;
      return { title: `${actor} regenerated ${target}` };
    }
    case "version_restored":
      return { title: `${actor} restored an itinerary version` };
    case "generation_job_failed": {
      const code = asString(metadata.errorCode);
      return {
        title: `${actor} had a generation job fail`,
        description: code ? `Reason: ${code}` : undefined
      };
    }
    case "comment_created":
      return { title: `${actor} commented on ${itemTarget(metadata)}` };
    case "comment_updated":
      return { title: `${actor} edited a comment on ${itemTarget(metadata)}` };
    case "comment_deleted":
      return { title: `${actor} deleted a comment on ${itemTarget(metadata)}` };
    case "trip_poll_created":
      return { title: `${actor} created a poll`, description: asString(metadata.pollTitle) };
    case "trip_poll_closed":
      return { title: `${actor} closed a poll`, description: asString(metadata.pollTitle) };
    case "trip_poll_archived":
      return { title: `${actor} archived a poll`, description: asString(metadata.pollTitle) };
    case "collaborator_invited": {
      const email = asString(metadata.collaboratorEmail);
      const role = asString(metadata.role);
      const who = email ?? "a collaborator";
      return { title: role ? `${actor} invited ${who} as ${role}` : `${actor} invited ${who}` };
    }
    case "collaborator_accepted":
      return { title: `${actor} accepted the invitation` };
    case "collaborator_declined":
      return { title: `${actor} declined the invitation` };
    case "collaborator_role_changed": {
      const oldRole = asString(metadata.oldRole);
      const newRole = asString(metadata.newRole);
      if (oldRole && newRole) {
        return { title: `${actor} changed a collaborator from ${oldRole} to ${newRole}` };
      }
      return { title: `${actor} changed a collaborator's role` };
    }
    case "collaborator_removed":
      return { title: `${actor} removed a collaborator` };
    case "share_created":
      return { title: `${actor} created a share link` };
    case "share_updated":
      return { title: `${actor} updated share settings` };
    case "share_disabled":
      return { title: `${actor} disabled the share link` };
    case "share_password_enabled":
      return { title: `${actor} enabled share password protection` };
    case "share_password_disabled":
      return { title: `${actor} disabled share password protection` };
    case "share_expiration_updated":
      return { title: `${actor} updated share expiration` };
    case "accommodation_added":
      return { title: `${actor} added ${accommodationLabel(metadata)}` };
    case "accommodation_updated":
      return { title: `${actor} updated ${accommodationLabel(metadata)}` };
    case "accommodation_removed":
      return { title: `${actor} removed ${accommodationLabel(metadata)}` };
    case "calendar_synced":
      return { title: `${actor} synced the trip calendar` };
    case "calendar_sync_removed":
      return { title: `${actor} removed calendar sync` };
    default:
      return { title: "Activity recorded" };
  }
}

function resolveActorLabel(
  actorUserId: string | null | undefined,
  currentUserId: string | null | undefined
): string {
  if (!actorUserId) {
    return "System";
  }
  if (currentUserId && actorUserId === currentUserId) {
    return "You";
  }
  return "Collaborator";
}

function dayLabel(metadata: Record<string, unknown>): string {
  const day = asNumber(metadata.dayNumber);
  return day != null ? `Day ${day}` : "a day";
}

// "Day 2 · Louvre Museum" when both are present; degrades gracefully otherwise.
function itemTarget(metadata: Record<string, unknown>): string {
  const day = asNumber(metadata.dayNumber);
  const name = asString(metadata.itemName);
  if (day != null && name) {
    return `Day ${day} · ${name}`;
  }
  if (day != null) {
    return `Day ${day}`;
  }
  if (name) {
    return name;
  }
  return "an item";
}

function accommodationLabel(metadata: Record<string, unknown>): string {
  return asString(metadata.name) ?? "the accommodation";
}

function asNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function asString(value: unknown): string | undefined {
  return typeof value === "string" && value.trim().length > 0 ? value.trim() : undefined;
}
