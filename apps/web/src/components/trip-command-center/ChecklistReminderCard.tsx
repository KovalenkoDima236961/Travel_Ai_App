import { ReadinessCard } from "./ReadinessCard";
import type { ReadinessCard as ReadinessCardModel } from "@/types/trip-command-center";

export function ChecklistReminderCard({ card }: { card: ReadinessCardModel }) {
  return <ReadinessCard card={card} />;
}
