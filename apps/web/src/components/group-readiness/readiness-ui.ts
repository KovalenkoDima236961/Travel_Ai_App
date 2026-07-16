import type { ReadinessCategory, ReadinessIssueSeverity, ReadinessLevel } from "@/types/group-readiness";

export const levelLabel: Record<ReadinessLevel, string> = {
  ready: "Ready",
  almost_ready: "Almost ready",
  needs_attention: "Needs attention",
  not_ready: "Not ready"
};

export function levelClasses(level: ReadinessLevel) {
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

export function severityClasses(severity: ReadinessIssueSeverity) {
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
      return "border-sand-300 bg-white text-cocoa-600";
  }
}

export function categoryLabel(category: ReadinessCategory) {
  switch (category) {
    case "availability":
      return "Availability";
    case "polls":
      return "Polls";
    case "checklist":
      return "Checklist";
    case "reminders":
      return "Reminders";
    case "settlements":
      return "Settlements";
    case "approval":
      return "Approval";
    case "activity":
      return "Activity";
    case "calendar":
      return "Calendar";
    case "expenses":
      return "Expenses";
    default:
      return category.replaceAll("_", " ");
  }
}

export function scoreBarClass(score: number) {
  if (score >= 90) return "bg-[#3E6B5A]";
  if (score >= 75) return "bg-[#7B8A3A]";
  if (score >= 50) return "bg-[#C28A3B]";
  return "bg-[#B3402E]";
}

