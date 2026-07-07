import { Card } from "@/shared/ui/card";
import {
  getPresenceDisplayName,
  getPresenceEditingWarning
} from "@/entities/presence/model";
import { cn } from "@/lib/utils";
import type { TripPresenceSnapshot } from "@/entities/presence/model";

type TripPresenceIndicatorProps = {
  snapshot: TripPresenceSnapshot | null;
  currentUserId?: string | null;
  isConnected?: boolean;
};

type PresenceEditingWarningProps = {
  snapshot: TripPresenceSnapshot | null;
  currentUserId?: string | null;
};

export function TripPresenceIndicator({
  snapshot,
  currentUserId,
  isConnected = false
}: TripPresenceIndicatorProps) {
  const users = snapshot?.users ?? [];
  const hasOnlyCurrentUser =
    users.length === 1 && Boolean(currentUserId) && users[0]?.userId === currentUserId;

  return (
    <Card>
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-lg font-semibold text-slate-950">Currently here</h2>
        <span
          className={cn(
            "h-2.5 w-2.5 rounded-full",
            isConnected ? "bg-emerald-500" : "bg-slate-300"
          )}
          title={isConnected ? "Presence connected" : "Presence reconnecting"}
        />
      </div>

      {users.length === 0 ? (
        <p className="mt-4 text-sm text-slate-500">Presence is connecting.</p>
      ) : hasOnlyCurrentUser ? (
        <p className="mt-4 text-sm text-slate-500">Only you are here.</p>
      ) : (
        <ul className="mt-4 space-y-3">
          {users.map((user) => (
            <li
              className="flex flex-wrap items-center justify-between gap-2 text-sm"
              key={user.userId}
            >
              <span className="font-medium text-slate-800">
                {getPresenceDisplayName(user, currentUserId)}
              </span>
              <span className="flex flex-wrap items-center gap-2">
                <PresenceBadge tone="role">{user.role}</PresenceBadge>
                <PresenceBadge tone={user.state === "editing" ? "editing" : "viewing"}>
                  {user.state}
                </PresenceBadge>
              </span>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}

export function PresenceEditingWarning({
  snapshot,
  currentUserId
}: PresenceEditingWarningProps) {
  const message = getPresenceEditingWarning(snapshot, currentUserId);
  if (!message) {
    return null;
  }
  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
      {message}
    </div>
  );
}

function PresenceBadge({
  children,
  tone
}: {
  children: string;
  tone: "role" | "viewing" | "editing";
}) {
  return (
    <span
      className={cn(
        "rounded-full px-2.5 py-1 text-xs font-medium capitalize",
        tone === "role" && "border border-slate-200 bg-slate-50 text-slate-700",
        tone === "viewing" && "border border-sky-200 bg-sky-50 text-sky-800",
        tone === "editing" && "border border-amber-300 bg-amber-100 text-amber-900"
      )}
    >
      {children}
    </span>
  );
}

export { getPresenceDisplayName, getPresenceEditingWarning } from "@/entities/presence/model";
