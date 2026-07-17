import Link from "next/link";
import { useTranslations } from "next-intl";
import type { CopilotSource } from "@/types/copilot";

export function CopilotSourceBadges({ sources, onNavigate }: { sources: CopilotSource[]; onNavigate?: () => void }) {
  const t = useTranslations("copilot");
  if (sources.length === 0) {
    return null;
  }
  return (
    <div className="mt-3">
      <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-cocoa-500">{t("sources")}</p>
      <div className="flex flex-wrap gap-1.5">
        {sources.map((source) => (
          <Link
            className="rounded-full border border-sand-400 bg-white px-2.5 py-1 text-xs font-medium text-cocoa-700 transition hover:border-cocoa-500"
            href={source.href}
            key={`${source.type}-${source.href}`}
            onClick={onNavigate}
          >
            {source.label}
          </Link>
        ))}
      </div>
    </div>
  );
}
