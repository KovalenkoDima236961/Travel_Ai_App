import type { RouteValidationWarning } from "@/entities/route/model";

type RouteValidationWarningsProps = {
  warnings: RouteValidationWarning[];
};

export function RouteValidationWarnings({ warnings }: RouteValidationWarningsProps) {
  if (warnings.length === 0) {
    return null;
  }

  return (
    <div className="rounded-[16px] border border-[#EAD9B8] bg-[#FDF0E3] p-4">
      <p className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#96682A]">
        Route warnings
      </p>
      <ul className="mt-2 space-y-1.5 text-[13.5px] leading-[1.45] text-[#7B5521]">
        {warnings.map((warning) => (
          <li key={`${warning.code}-${warning.message}`}>{warning.message}</li>
        ))}
      </ul>
    </div>
  );
}
