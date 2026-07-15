import { transportModeLabel } from "@/components/routes/route-options";
import type { TransportOption } from "@/types/transport";
import { formatTransportDuration, formatTransportPrice } from "./transport-format";

type Props = {
  options: TransportOption[];
};

export function CompareTransportModesTable({ options }: Props) {
  const rows = summarizeByMode(options);
  if (rows.length === 0) {
    return null;
  }
  return (
    <div className="overflow-x-auto rounded-lg border border-sand-300">
      <table className="min-w-full divide-y divide-sand-200 text-left text-[12.5px]">
        <thead className="bg-sand-100 text-cocoa-500">
          <tr>
            <th className="px-3 py-2 font-semibold">Mode</th>
            <th className="px-3 py-2 font-semibold">Fastest</th>
            <th className="px-3 py-2 font-semibold">Lowest price</th>
            <th className="px-3 py-2 font-semibold">Options</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-sand-200 bg-white text-cocoa-700">
          {rows.map((row) => (
            <tr key={row.mode}>
              <td className="px-3 py-2 font-semibold">{transportModeLabel(row.mode)}</td>
              <td className="px-3 py-2">{formatTransportDuration(row.fastestMinutes)}</td>
              <td className="px-3 py-2">{formatTransportPrice(row.lowestPrice)}</td>
              <td className="px-3 py-2">{row.count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function summarizeByMode(options: TransportOption[]) {
  const byMode = new Map<string, TransportOption[]>();
  for (const option of options) {
    byMode.set(option.mode, [...(byMode.get(option.mode) ?? []), option]);
  }
  return Array.from(byMode.entries())
    .map(([mode, modeOptions]) => {
      const priced = modeOptions.filter((option) => option.estimatedPrice);
      const lowest = priced.sort(
        (a, b) => (a.estimatedPrice?.amount ?? Number.MAX_SAFE_INTEGER) - (b.estimatedPrice?.amount ?? Number.MAX_SAFE_INTEGER)
      )[0]?.estimatedPrice;
      return {
        mode,
        count: modeOptions.length,
        fastestMinutes: Math.min(...modeOptions.map((option) => option.durationMinutes || Number.MAX_SAFE_INTEGER)),
        lowestPrice: lowest ?? null
      };
    })
    .sort((a, b) => a.fastestMinutes - b.fastestMinutes);
}
