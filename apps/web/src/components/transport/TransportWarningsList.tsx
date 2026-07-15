type Props = {
  warnings?: string[] | null;
};

export function TransportWarningsList({ warnings }: Props) {
  const clean = (warnings ?? []).map((warning) => warning.trim()).filter(Boolean);
  if (clean.length === 0) {
    return null;
  }
  return (
    <ul className="flex flex-col gap-1 text-[12.5px] text-amber-700">
      {clean.map((warning) => (
        <li key={warning}>- {warning}</li>
      ))}
    </ul>
  );
}
