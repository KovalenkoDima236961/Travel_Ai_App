type OnboardingProgressProps = {
  current: number;
  total: number;
  label: string;
};

export function OnboardingProgress({ current, total, label }: OnboardingProgressProps) {
  const percentage = Math.round((current / total) * 100);
  return (
    <div aria-label={label} role="progressbar" aria-valuemin={1} aria-valuemax={total} aria-valuenow={current}>
      <div className="flex items-center justify-between text-[12.5px] font-semibold text-cocoa-500">
        <span>{label}</span>
        <span>{percentage}%</span>
      </div>
      <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-sand-200">
        <div
          className="h-full rounded-full bg-[#3E6B5A] transition-all"
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}
