import type { ReadinessCardStatus } from "@/types/trip-command-center";
import type { TripHealthIssueSeverity } from "@/types/trip-health";

export const readinessStatusLabel: Record<ReadinessCardStatus, string> = {
  ready: "Ready",
  almost_ready: "Almost ready",
  needs_attention: "Needs attention",
  blocked: "Blocked",
  empty: "Empty",
  unavailable: "Unavailable"
};

export function readinessStatusClasses(status: ReadinessCardStatus) {
  switch (status) {
    case "ready":
      return "border-[#CFE3D3] bg-[#EFF7F1] text-[#2F5C3C]";
    case "almost_ready":
      return "border-[#DCE7C8] bg-[#F5F8EA] text-[#5E6B2D]";
    case "needs_attention":
      return "border-[#EAD9B8] bg-[#FDF0E3] text-[#96682A]";
    case "blocked":
      return "border-[#E5C3B6] bg-[#FBF0EB] text-[#B3402E]";
    case "empty":
      return "border-[#D8C8B2] bg-sand-50 text-cocoa-500";
    case "unavailable":
      return "border-[#D6DEE8] bg-[#F4F7FA] text-[#536171]";
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
      return "border-sand-300 bg-white text-cocoa-600";
  }
}
