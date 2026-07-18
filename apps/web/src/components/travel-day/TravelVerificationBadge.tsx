import type { TravelDaySummary } from "@/types/travel-day";
export function TravelVerificationBadge({ verification }: { verification: TravelDaySummary["verification"] }) { return <span className="inline-flex rounded-full border border-sand-300 bg-white px-3 py-1.5 text-xs font-semibold text-cocoa-600">Verification {verification.unavailable ? "unavailable" : `${verification.score}/100`}</span>; }
