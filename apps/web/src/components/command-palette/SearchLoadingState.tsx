"use client";

type SearchLoadingStateProps = {
  label: string;
};

export function SearchLoadingState({ label }: SearchLoadingStateProps) {
  return (
    <div className="flex items-center gap-3 px-5 py-4 text-sm text-slate-500">
      <span className="h-4 w-4 animate-spin rounded-full border-2 border-slate-300 border-t-slate-700" />
      <span>{label}</span>
    </div>
  );
}
