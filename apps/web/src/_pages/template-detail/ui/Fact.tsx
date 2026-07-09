type FactProps = {
  label: string;
  value: string;
};

// Warm stat card for the redesigned Template Detail hero (Destination / Duration
// / Estimate / Updated). Slice-local — only the template-detail page renders it.
export function Fact({ label, value }: FactProps) {
  return (
    <div className="rounded-2xl border border-sand-300 bg-white px-5 py-[18px]">
      <p className="text-[12px] font-semibold uppercase tracking-[0.05em] text-[#A08D78]">
        {label}
      </p>
      <p className="mt-2 break-words text-[16px] font-semibold text-cocoa-900">{value}</p>
    </div>
  );
}
