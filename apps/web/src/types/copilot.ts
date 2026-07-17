export type CopilotActionType =
  | "open_command_center"
  | "open_trip_health"
  | "open_route"
  | "open_route_leg"
  | "find_transport"
  | "open_budget"
  | "open_budget_confidence"
  | "open_expenses"
  | "upload_receipt"
  | "add_expense"
  | "open_checklist"
  | "generate_checklist_screen"
  | "open_reminders"
  | "open_group_readiness"
  | "request_availability_screen"
  | "open_polls"
  | "open_approval"
  | "open_policy"
  | "open_itinerary"
  | "open_itinerary_day"
  | "open_generation_quality"
  | "open_version_history"
  | "open_share_settings"
  | "open_offline_settings"
  | "open_notification_settings"
  | "open_settings"
  | "open_search"
  | "no_action";

export type CopilotSourceType =
  | "command_center"
  | "trip_health"
  | "budget_confidence"
  | "group_readiness"
  | "route_summary"
  | "itinerary_summary"
  | "checklist_summary"
  | "reminders_summary"
  | "expenses_summary"
  | "approval_status"
  | "policy_evaluation"
  | "generation_quality"
  | "personalization"
  | "notification_summary"
  | "app_help"
  | "unknown";

export type CopilotAction = {
  type: CopilotActionType;
  label: string;
  href: string;
  style: "primary" | "secondary";
};

export type CopilotSource = {
  type: CopilotSourceType;
  label: string;
  href: string;
};

export type CopilotClientContext = {
  currentTab?: string;
  currentPath?: string;
  selectedIssueId?: string;
  selectedDayNumber?: number;
  selectedRouteLegId?: string;
};

export type CopilotRequest = {
  conversationId?: string;
  message: string;
  clientContext?: CopilotClientContext;
};

export type CopilotResponse = {
  conversationId: string;
  messageId: string;
  answer: string;
  actions: CopilotAction[];
  sources: CopilotSource[];
  warnings: string[];
  permissionNotes: string[];
  metadata: {
    mode: "mock" | "ai";
    intent: string;
    safeContextUsed: string[];
  };
};

export type CopilotMessage = {
  id: string;
  role: "user" | "assistant";
  content: string;
  response?: CopilotResponse;
};

export type CopilotSuggestedPrompt = {
  id: string;
  label: string;
};

export type CopilotConversation = {
  id: string | null;
  messages: CopilotMessage[];
};
