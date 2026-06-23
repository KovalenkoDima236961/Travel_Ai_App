import { Card } from "@/components/ui/Card";

export function SettingsSkeleton() {
  return (
    <div className="space-y-6" aria-label="Loading settings">
      <SkeletonCard />
      <SkeletonCard />
    </div>
  );
}

function SkeletonCard() {
  return (
    <Card className="animate-pulse">
      <div className="h-5 w-40 rounded bg-slate-200" />
      <div className="mt-6 grid gap-5 md:grid-cols-2">
        <div className="h-16 rounded bg-slate-100" />
        <div className="h-16 rounded bg-slate-100" />
        <div className="h-16 rounded bg-slate-100" />
        <div className="h-16 rounded bg-slate-100" />
      </div>
      <div className="mt-6 h-11 w-28 rounded bg-slate-200" />
    </Card>
  );
}
