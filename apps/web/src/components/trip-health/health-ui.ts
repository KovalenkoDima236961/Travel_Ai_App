import type {
  TripHealthCategory,
  TripHealthIssueSeverity,
  TripHealthLevel
} from "@/types/trip-health";

export const severityRank: Record<TripHealthIssueSeverity, number> = {
  critical: 4,
  high: 3,
  warning: 2,
  info: 1
};

export const severityLabel: Record<TripHealthIssueSeverity, string> = {
  critical: "Critical",
  high: "High",
  warning: "Warning",
  info: "Info"
};

export const categoryLabel: Record<TripHealthCategory, string> = {
  itinerary: "Itinerary",
  route: "Route",
  transport: "Transport",
  budget: "Budget",
  availability: "Availability",
  collaboration: "Collaboration",
  checklist: "Checklist",
  reminders: "Reminders",
  accommodation: "Accommodation",
  expenses: "Expenses",
  policy: "Policy",
  approval: "Approval",
  offline: "Offline",
  data_quality: "Data quality",
  public_share: "Public share",
  other: "Other"
};

export const levelLabel: Record<TripHealthLevel, string> = {
  ready: "Ready",
  almost_ready: "Almost ready",
  needs_attention: "Needs attention",
  not_ready: "Not ready"
};

export function levelClasses(level: TripHealthLevel) {
  switch (level) {
    case "ready":
      return "border-[#CFE3D3] bg-[#EFF7F1] text-[#2F5C3C]";
    case "almost_ready":
      return "border-[#DCE7C8] bg-[#F5F8EA] text-[#5E6B2D]";
    case "needs_attention":
      return "border-[#EAD9B8] bg-[#FDF0E3] text-[#96682A]";
    case "not_ready":
      return "border-[#E5C3B6] bg-[#FBF0EB] text-[#B3402E]";
    default:
      return "border-sand-300 bg-white text-cocoa-700";
  }
}

export function severityClasses(severity: TripHealthIssueSeverity) {
  switch (severity) {
    case "critical":
      return "border-[#DDAEA0] bg-[#FBF0EB] text-[#A93624]";
    case "high":
      return "border-[#E8C3A3] bg-[#FFF3E8] text-[#9B571F]";
    case "warning":
      return "border-[#EAD9B8] bg-[#FDF7E8] text-[#846326]";
    case "info":
      return "border-[#D6DEE8] bg-[#F4F7FA] text-[#536171]";
    default:
      return "border-sand-300 bg-white text-cocoa-500";
  }
}

export function scoreBarClass(score: number) {
  if (score >= 90) {
    return "bg-[#4D7C5A]";
  }
  if (score >= 75) {
    return "bg-[#8A9B45]";
  }
  if (score >= 50) {
    return "bg-[#C28A3A]";
  }
  return "bg-[#B3402E]";
}

export function formatGeneratedAt(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit"
  }).format(date);
}
