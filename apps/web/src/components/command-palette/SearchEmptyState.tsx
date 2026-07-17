"use client";

type SearchEmptyStateProps = {
  title: string;
  description: string;
};

export function SearchEmptyState({ title, description }: SearchEmptyStateProps) {
  return (
    <div className="px-6 py-10 text-center">
      <p className="text-sm font-semibold text-slate-800">{title}</p>
      <p className="mx-auto mt-2 max-w-sm text-sm leading-6 text-slate-500">{description}</p>
    </div>
  );
}
