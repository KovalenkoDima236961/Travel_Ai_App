import { buildTripWorkspaceHref } from "./navigation";

export type TripWorkspaceActionId =
  | "open_today"
  | "edit_itinerary"
  | "invite_people"
  | "share_trip"
  | "export_trip"
  | "download_offline"
  | "save_template"
  | "view_versions"
  | "view_recap"
  | "view_analytics"
  | "open_search"
  | "archive_trip"
  | "restore_trip";

export type TripWorkspaceActionGroup = "primary" | "share_export" | "history" | "reuse" | "settings";

export type TripWorkspaceActionContext = {
  tripId: string;
  role: "owner" | "editor" | "viewer";
  canEdit: boolean;
  canManageCollaborators: boolean;
  canManageShare: boolean;
  canRestoreVersion: boolean;
  canArchive: boolean;
  archived: boolean;
  completed: boolean;
  online: boolean;
  hasItinerary: boolean;
  flags: Partial<Record<
    | "public_sharing_enabled"
    | "data_exports_enabled"
    | "offline_mode_enabled"
    | "trip_workspace_shared_actions_enabled",
    boolean
  >>;
};

export type TripWorkspaceAction = {
  id: TripWorkspaceActionId;
  labelKey: TripWorkspaceActionId;
  group: TripWorkspaceActionGroup;
  href?: string;
  command?: "open_search" | "edit_itinerary" | "save_template" | "export_trip" | "archive_trip" | "restore_trip";
  destructive?: boolean;
  disabledReason?: "offline" | "read_only" | "archived";
  analyticsEvent: "trip_workspace_quick_action_used";
};

export function getTripWorkspaceActions(
  context: TripWorkspaceActionContext
): TripWorkspaceAction[] {
  const action = (
    input: Omit<TripWorkspaceAction, "analyticsEvent">
  ): TripWorkspaceAction => ({
    ...input,
    analyticsEvent: "trip_workspace_quick_action_used"
  });
  const actions: TripWorkspaceAction[] = [
    action({
      id: "open_today",
      labelKey: "open_today",
      group: "primary",
      href: `/trips/${context.tripId}/today`
    }),
    action({
      id: "open_search",
      labelKey: "open_search",
      group: "primary",
      command: "open_search"
    }),
    action({
      id: "view_recap",
      labelKey: "view_recap",
      group: "history",
      href: `/trips/${context.tripId}/recap`
    }),
    action({
      id: "view_analytics",
      labelKey: "view_analytics",
      group: "history",
      href: `/trips/${context.tripId}/analytics`
    })
  ];

  if (context.canEdit && context.hasItinerary && !context.archived) {
    actions.push(action({
      id: "edit_itinerary",
      labelKey: "edit_itinerary",
      group: "primary",
      command: "edit_itinerary",
      disabledReason: context.online ? undefined : "offline"
    }));
  }
  if (context.canManageCollaborators && !context.archived) {
    actions.push(action({
      id: "invite_people",
      labelKey: "invite_people",
      group: "share_export",
      href: buildTripWorkspaceHref(context.tripId, "group", { view: "people", action: "invite" }),
      disabledReason: context.online ? undefined : "offline"
    }));
  }
  if (context.canManageShare && context.flags.public_sharing_enabled) {
    actions.push(action({
      id: "share_trip",
      labelKey: "share_trip",
      group: "share_export",
      href: buildTripWorkspaceHref(context.tripId, "more", { view: "sharing" }),
      disabledReason: context.online ? undefined : "offline"
    }));
  }
  if (context.flags.data_exports_enabled) {
    actions.push(action({
      id: "export_trip",
      labelKey: "export_trip",
      group: "share_export",
      command: "export_trip"
    }));
  }
  if (context.flags.offline_mode_enabled) {
    actions.push(action({
      id: "download_offline",
      labelKey: "download_offline",
      group: "share_export",
      href: buildTripWorkspaceHref(context.tripId, "prepare", { view: "offline" })
    }));
  }
  if (context.canRestoreVersion && context.hasItinerary) {
    actions.push(action({
      id: "view_versions",
      labelKey: "view_versions",
      group: "history",
      href: buildTripWorkspaceHref(context.tripId, "more", { view: "versions" })
    }));
  }
  if (context.canEdit && context.completed && context.hasItinerary) {
    actions.push(action({
      id: "save_template",
      labelKey: "save_template",
      group: "reuse",
      command: "save_template",
      disabledReason: context.online ? undefined : "offline"
    }));
  }
  if (context.canArchive && !context.archived) {
    actions.push(action({
      id: "archive_trip",
      labelKey: "archive_trip",
      group: "settings",
      command: "archive_trip",
      destructive: true,
      disabledReason: context.online ? undefined : "offline"
    }));
  }
  if (context.canArchive && context.archived) {
    actions.push(action({
      id: "restore_trip",
      labelKey: "restore_trip",
      group: "settings",
      command: "restore_trip",
      disabledReason: context.online ? undefined : "offline"
    }));
  }

  return actions;
}
