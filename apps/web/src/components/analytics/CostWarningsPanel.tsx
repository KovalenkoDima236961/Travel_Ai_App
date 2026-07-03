import { Card } from "@/components/ui/Card";

type CostWarningsPanelProps = {
  warnings: string[];
};

export function CostWarningsPanel({ warnings }: CostWarningsPanelProps) {
  if (warnings.length === 0) {
    return null;
  }

  return (
    <Card className="border-amber-200 bg-amber-50">
      <h2 className="text-lg font-semibold text-amber-950">Warnings</h2>
      <ul className="mt-3 list-disc space-y-2 pl-5 text-sm leading-6 text-amber-900">
        {Array.from(new Set(warnings)).map((warning) => (
          <li key={warning}>{warning}</li>
        ))}
      </ul>
    </Card>
  );
}
