import type { AppNotification } from "@/entities/notification/model";

const NOTIFICATION_TITLE_KEYS: Partial<Record<AppNotification["type"], string>> = {
  workspace_invited: "workspaceInvited",
  comment_created: "commentCreated",
  workspace_budget_exceeded: "budgetExceeded",
  workspace_role_changed: "roleChanged",
  collaborator_role_changed: "roleChanged",
  collaboration_invited: "collaboratorInvited"
};

export function localizedNotificationTitle(
  notification: AppNotification,
  translate: (key: string) => string
): string {
  const key = NOTIFICATION_TITLE_KEYS[notification.type];
  return key ? translate(key) : notification.title;
}
