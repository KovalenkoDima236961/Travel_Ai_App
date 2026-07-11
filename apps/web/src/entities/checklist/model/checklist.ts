export type ChecklistStatus = "active" | "archived";

export type ChecklistCategory =
  | "documents"
  | "clothing"
  | "electronics"
  | "health_safety"
  | "transport"
  | "accommodation"
  | "activities"
  | "food_water"
  | "money"
  | "before_departure"
  | "group_items"
  | "camping_hiking"
  | "weather"
  | "other";

export type ChecklistPriority = "low" | "medium" | "high" | "critical";

export type ChecklistSource = "ai" | "manual" | "template" | "regenerated" | "system";

export type ChecklistItemType =
  | "packing"
  | "preparation"
  | "booking_check"
  | "document"
  | "shared_group_item"
  | "reminder"
  | "safety_check"
  | "other";

export type GenerateChecklistMode = "full" | "add_missing" | "category";

export const CHECKLIST_CATEGORIES: ChecklistCategory[] = [
  "documents",
  "clothing",
  "electronics",
  "health_safety",
  "transport",
  "accommodation",
  "activities",
  "food_water",
  "money",
  "before_departure",
  "group_items",
  "camping_hiking",
  "weather",
  "other"
];

export const CHECKLIST_PRIORITIES: ChecklistPriority[] = [
  "critical",
  "high",
  "medium",
  "low"
];

export const CHECKLIST_ITEM_TYPES: ChecklistItemType[] = [
  "packing",
  "preparation",
  "booking_check",
  "document",
  "shared_group_item",
  "reminder",
  "safety_check",
  "other"
];

export type TripChecklistItem = {
  id: string;
  checklistId: string;
  title: string;
  description?: string | null;
  category: ChecklistCategory;
  itemType: ChecklistItemType;
  priority: ChecklistPriority;
  quantity?: number | null;
  assignedToUserId?: string | null;
  assignedToDisplayName?: string | null;
  dueDate?: string | null;
  checked: boolean;
  checkedAt?: string | null;
  checkedByUserId?: string | null;
  source: ChecklistSource;
  reason?: string | null;
  relatedDayNumber?: number | null;
  relatedItemIndex?: number | null;
  relatedItemId?: string | null;
  sortOrder: number;
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
};

export type TripChecklist = {
  id: string;
  tripId: string;
  status: ChecklistStatus;
  generatedFromRevision?: number | null;
  generatedFromItineraryRevision?: number | null;
  generatedFromRouteRevision?: number | null;
  title: string;
  summary?: string | null;
  createdByUserId: string;
  updatedAt: string;
  items: TripChecklistItem[];
  metadata?: Record<string, unknown>;
  createdAt: string;
};

export type ChecklistCategorySummary = {
  category: ChecklistCategory;
  total: number;
  checked: number;
};

export type ChecklistSummary = {
  totalItems: number;
  checkedItems: number;
  uncheckedItems: number;
  highPriorityUnchecked: number;
  assignedToMe: number;
  categories: ChecklistCategorySummary[];
};

export type ChecklistViewResponse = {
  checklist: TripChecklist | null;
  summary?: ChecklistSummary | null;
  canGenerate: boolean;
};

export type GenerateChecklistRequest = {
  mode?: GenerateChecklistMode;
  categories?: ChecklistCategory[];
  instructions?: string;
  preserveCheckedItems?: boolean;
  preserveManualItems?: boolean;
  replaceAiItems?: boolean;
  outputLanguage?: "en" | "es" | "uk" | "fr";
};

export type ChecklistItemPayload = {
  title: string;
  description?: string | null;
  category: ChecklistCategory;
  itemType?: ChecklistItemType;
  priority?: ChecklistPriority;
  quantity?: number | null;
  assignedToUserId?: string | null;
  dueDate?: string | null;
  reason?: string | null;
  relatedDayNumber?: number | null;
  relatedItemIndex?: number | null;
  relatedItemId?: string | null;
  metadata?: Record<string, unknown>;
};

export type UpdateChecklistItemPayload = Partial<ChecklistItemPayload> & {
  clearDescription?: boolean;
  clearQuantity?: boolean;
  clearAssignee?: boolean;
  clearDueDate?: boolean;
  clearReason?: boolean;
  clearRelatedDay?: boolean;
  clearRelatedIndex?: boolean;
  clearRelatedItem?: boolean;
  sortOrder?: number;
};

