import { ExclamationIcon } from "./icons";

type WarningsPanelProps = {
  warnings: string[];
};

/**
 * Slice-local restyle of the shared CostWarningsPanel. The mock omits it, but the
 * real analytics response can surface data-quality warnings the user should see.
 */
export function WarningsPanel({ warnings }: WarningsPanelProps) {
  if (warnings.length === 0) {
    return null;
  }

  return (
    <div className="rounded-[20px] border border-[#EFD9B8] bg-[#FFFDF7] px-7 py-6">
      <div className="flex items-center gap-2.5">
        <ExclamationIcon className="h-5 w-5 text-[#96682A]" />
        <h2 className="text-[15.5px] font-semibold text-cocoa-900">Warnings</h2>
      </div>
      <ul className="mt-3 list-disc space-y-2 pl-8 text-sm leading-[1.55] text-cocoa-500">
        {Array.from(new Set(warnings)).map((warning) => (
          <li key={warning}>{warning}</li>
        ))}
      </ul>
    </div>
  );
}
