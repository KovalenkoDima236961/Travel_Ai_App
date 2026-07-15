import { categoryLabel } from "./health-ui";
import type { TripHealthCategory } from "@/types/trip-health";

const categoryTone: Partial<Record<TripHealthCategory, string>> = {
  policy: "bg-[#FBF0EB] text-[#A93624]",
  approval: "bg-[#FBF0EB] text-[#A93624]",
  itinerary: "bg-[#F6EEE4] text-[#7E5436]",
  route: "bg-[#EEF4F0] text-[#3E6B5A]",
  transport: "bg-[#EEF4F0] text-[#3E6B5A]",
  budget: "bg-[#FDF7E8] text-[#846326]",
  expenses: "bg-[#FDF7E8] text-[#846326]",
  availability: "bg-[#F4F7FA] text-[#536171]",
  checklist: "bg-[#F4F1F8] text-[#655278]",
  reminders: "bg-[#F4F1F8] text-[#655278]",
  accommodation: "bg-[#F6EEE4] text-[#7E5436]",
  data_quality: "bg-[#F4F7FA] text-[#536171]"
};

export function HealthCategoryIcon({ category }: { category: TripHealthCategory }) {
  const label = categoryLabel[category];
  return (
    <span
      aria-hidden
      className={`inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-[12px] font-semibold ${
        categoryTone[category] ?? "bg-sand-200 text-cocoa-600"
      }`}
      title={label}
    >
      {label.slice(0, 1)}
    </span>
  );
}
