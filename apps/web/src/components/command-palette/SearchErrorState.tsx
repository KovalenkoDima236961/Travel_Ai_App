"use client";

type SearchErrorStateProps = {
  title: string;
  description: string;
};

export function SearchErrorState({ title, description }: SearchErrorStateProps) {
  return (
    <div className="px-5 py-4 text-sm">
      <p className="font-semibold text-red-700">{title}</p>
      <p className="mt-1 text-red-600">{description}</p>
    </div>
  );
}
