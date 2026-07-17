import type { ReactNode } from "react";

type FeatureHintProps = {
  title: string;
  description: string;
  action?: ReactNode;
};

export function FeatureHint({ title, description, action }: FeatureHintProps) {
  return (
    <aside className="rounded-[14px] border border-[#DCE8DD] bg-[#F2F7F1] px-4 py-3.5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-[13.5px] font-semibold text-[#38543F]">{title}</p>
          <p className="mt-1 text-[13px] leading-[1.55] text-[#58705E]">{description}</p>
        </div>
        {action}
      </div>
    </aside>
  );
}
