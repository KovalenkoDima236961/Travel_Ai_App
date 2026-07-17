import Link from "next/link";
import { useTranslations } from "next-intl";
import { buttonStyles } from "@/shared/ui/button";
import type { CopilotAction } from "@/types/copilot";

export function CopilotActionButtons({
  actions,
  onNavigate
}: {
  actions: CopilotAction[];
  onNavigate?: () => void;
}) {
  const t = useTranslations("copilot");
  if (actions.length === 0) {
    return null;
  }
  return (
    <div className="mt-3">
      <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-cocoa-500">
        {t("suggestedActions")}
      </p>
      <div className="flex flex-wrap gap-2">
        {actions.map((action) => (
          <Link
            className={buttonStyles({
              variant: action.style === "primary" ? "primary" : "secondary",
              size: "sm"
            })}
            href={action.href}
            key={`${action.type}-${action.href}`}
            onClick={onNavigate}
          >
            {action.label}
          </Link>
        ))}
      </div>
    </div>
  );
}
