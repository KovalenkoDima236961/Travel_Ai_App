import { ReadinessCard } from "./ReadinessCard";
import type { ReadinessCard as ReadinessCardModel } from "@/types/trip-command-center";

export function OfflineStatusCard({
  card,
  onSyncNow,
  syncing = false
}: {
  card: ReadinessCardModel;
  onSyncNow?: () => void;
  syncing?: boolean;
}) {
  return (
    <div id="offline" className="scroll-mt-24">
      <ReadinessCard
        card={{
          ...card,
          primaryAction:
            onSyncNow && card.primaryAction?.label === "Sync now"
              ? null
              : card.primaryAction
        }}
      />
      {onSyncNow && card.primaryAction?.label === "Sync now" ? (
        <button
          type="button"
          onClick={onSyncNow}
          disabled={syncing}
          className="mt-3 inline-flex h-9 items-center rounded-full bg-cocoa-900 px-4 text-[13px] font-semibold text-sand-100 transition hover:bg-cocoa-700 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {syncing ? "Syncing..." : "Sync now"}
        </button>
      ) : null}
    </div>
  );
}
