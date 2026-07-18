import { RealWorldReadinessCard } from "./RealWorldReadinessCard";
import type { RealWorldReadiness } from "@/types/verification";

export function VerificationPanel({ readiness }: { readiness?: RealWorldReadiness | null }) {
  if (!readiness) {
    return null;
  }
  return <RealWorldReadinessCard readiness={readiness} sectionId="verification" />;
}
