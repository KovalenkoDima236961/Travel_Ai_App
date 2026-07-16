import type { ComponentType } from "react";
import type { NotificationType } from "@/entities/notification/model";
import {
  ChatBubbleIcon,
  CheckCircleIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  SparklesIcon,
  UserGroupIcon
} from "./icons";

type IconComponent = ComponentType<{ className?: string }>;

export type NotificationVisual = {
  Icon: IconComponent;
  /** Tailwind classes for the 40×40 rounded icon tile (background + icon color). */
  tileClassName: string;
};

// The design mock only shows a handful of tints; the real feed carries 23
// notification types, so each maps to one of four on-brand tiles. The green and
// amber icon tints and the alert tile are genuine one-offs (arbitrary values),
// matching the redesign convention of reserving tokens for the shared palette.
const CLAY = "bg-clay-tint text-clay-dark";
const GREEN = "bg-[#EDF3EA] text-[#2F7A57]";
const AMBER = "bg-[#FDF0E3] text-[#B57F24]";
const ALERT = "bg-[#F8E1DA] text-[#B23D22]";

const VISUALS: Record<NotificationType, NotificationVisual> = {
  // AI / itinerary activity.
  itinerary_generated: { Icon: SparklesIcon, tileClassName: CLAY },
  itinerary_updated: { Icon: SparklesIcon, tileClassName: CLAY },
  day_regenerated: { Icon: SparklesIcon, tileClassName: CLAY },
  item_regenerated: { Icon: SparklesIcon, tileClassName: CLAY },
  version_restored: { Icon: SparklesIcon, tileClassName: CLAY },
  budget_optimization_ready: { Icon: SparklesIcon, tileClassName: CLAY },

  // Conversation.
  comment_created: { Icon: ChatBubbleIcon, tileClassName: GREEN },
  trip_poll_created: { Icon: ChatBubbleIcon, tileClassName: GREEN },
  trip_poll_closed: { Icon: ChatBubbleIcon, tileClassName: GREEN },

  // Personal / trip collaboration.
  collaboration_invited: { Icon: UserGroupIcon, tileClassName: GREEN },
  collaboration_accepted: { Icon: UserGroupIcon, tileClassName: GREEN },
  collaborator_role_changed: { Icon: UserGroupIcon, tileClassName: GREEN },
  collaborator_removed: { Icon: UserGroupIcon, tileClassName: GREEN },
  group_readiness_nudge: { Icon: UserGroupIcon, tileClassName: GREEN },
  availability_nudge: { Icon: UserGroupIcon, tileClassName: GREEN },
  checklist_assignment_nudge: { Icon: UserGroupIcon, tileClassName: GREEN },
  reminder_task_nudge: { Icon: UserGroupIcon, tileClassName: GREEN },
  poll_vote_nudge: { Icon: UserGroupIcon, tileClassName: GREEN },
  settlement_nudge: { Icon: UserGroupIcon, tileClassName: GREEN },

  // Workspace membership.
  workspace_invited: { Icon: UserGroupIcon, tileClassName: CLAY },
  workspace_invitation_accepted: { Icon: UserGroupIcon, tileClassName: CLAY },
  workspace_invitation_declined: { Icon: UserGroupIcon, tileClassName: CLAY },
  workspace_member_removed: { Icon: UserGroupIcon, tileClassName: CLAY },
  workspace_role_changed: { Icon: UserGroupIcon, tileClassName: CLAY },
  workspace_trip_created: { Icon: UserGroupIcon, tileClassName: CLAY },

  // Budgets / approvals.
  workspace_budget_created: { Icon: CheckCircleIcon, tileClassName: AMBER },
  workspace_budget_updated: { Icon: CheckCircleIcon, tileClassName: AMBER },
  workspace_budget_archived: { Icon: CheckCircleIcon, tileClassName: AMBER },
  workspace_budget_nearing_limit: { Icon: ExclamationTriangleIcon, tileClassName: ALERT },
  workspace_budget_exceeded: { Icon: ExclamationTriangleIcon, tileClassName: ALERT },
  expense_added: { Icon: CheckCircleIcon, tileClassName: AMBER },
  settlement_paid: { Icon: CheckCircleIcon, tileClassName: AMBER },

  // Failures.
  generation_job_failed: { Icon: ExclamationTriangleIcon, tileClassName: ALERT },
  budget_optimization_failed: { Icon: ExclamationTriangleIcon, tileClassName: ALERT }
};

const FALLBACK: NotificationVisual = { Icon: ClockIcon, tileClassName: AMBER };

export function notificationVisual(type: NotificationType): NotificationVisual {
  return VISUALS[type] ?? FALLBACK;
}
