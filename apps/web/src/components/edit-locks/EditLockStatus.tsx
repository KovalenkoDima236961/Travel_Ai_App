import type { EditLockView } from "@/types/edit-locks";

type EditLockStatusProps = {
  lock: EditLockView | null;
};

export function EditLockStatus({ lock }: EditLockStatusProps) {
  if (!lock?.locked) {
    return null;
  }

  const label = lock.lockedByCurrentUser
    ? "You are editing this itinerary"
    : `${lock.lockedByDisplayName?.trim() || "A collaborator"} is editing this itinerary`;

  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm font-medium text-amber-900">
      {label}
    </div>
  );
}
