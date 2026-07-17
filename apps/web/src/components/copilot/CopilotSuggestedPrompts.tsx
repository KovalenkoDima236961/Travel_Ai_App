import { useTranslations } from "next-intl";
import type { CopilotSuggestedPrompt } from "@/types/copilot";

export function CopilotSuggestedPrompts({ prompts, onSelect }: { prompts: CopilotSuggestedPrompt[]; onSelect: (prompt: string) => void }) {
  const t = useTranslations("copilot");
  return (
    <section>
      <p className="mb-2 text-sm font-semibold text-cocoa-700">{t("suggestions")}</p>
      <div className="flex flex-wrap gap-2">
        {prompts.map((prompt) => (
          <button
            className="rounded-full border border-sand-400 bg-white px-3 py-1.5 text-left text-xs text-cocoa-700 transition hover:border-cocoa-500 hover:bg-sand-100"
            key={prompt.id}
            onClick={() => onSelect(prompt.label)}
            type="button"
          >
            {prompt.label}
          </button>
        ))}
      </div>
    </section>
  );
}
