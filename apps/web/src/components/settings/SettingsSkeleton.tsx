import { SettingsCard } from "@/components/settings/controls";

export function SettingsSkeleton() {
  return (
    <div className="flex flex-col gap-5" aria-label="Loading settings">
      <SkeletonCard />
      <SkeletonCard />
    </div>
  );
}

function SkeletonCard() {
  return (
    <SettingsCard className="animate-pulse">
      <div className="h-6 w-40 rounded bg-sand-300" />
      <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="h-16 rounded-xl bg-sand-200" />
        <div className="h-16 rounded-xl bg-sand-200" />
        <div className="h-16 rounded-xl bg-sand-200" />
        <div className="h-16 rounded-xl bg-sand-200" />
      </div>
      <div className="mt-6 flex justify-end">
        <div className="h-11 w-32 rounded-full bg-sand-300" />
      </div>
    </SettingsCard>
  );
}
