import { CopilotActionButtons } from "./CopilotActionButtons";
import { CopilotSourceBadges } from "./CopilotSourceBadges";
import type { CopilotMessage } from "@/types/copilot";

export function CopilotMessageBubble({ message, onNavigate }: { message: CopilotMessage; onNavigate?: () => void }) {
  const user = message.role === "user";
  return (
    <article className={user ? "ml-8" : "mr-3"}>
      <div
        className={
          user
            ? "rounded-2xl rounded-br-md bg-cocoa-900 px-3.5 py-3 text-sm leading-6 text-white"
            : "rounded-2xl rounded-bl-md border border-sand-300 bg-sand-50 px-3.5 py-3 text-sm leading-6 text-cocoa-700"
        }
      >
        <p>{message.content}</p>
        {!user && message.response ? (
          <>
            <CopilotActionButtons actions={message.response.actions} onNavigate={onNavigate} />
            <CopilotSourceBadges sources={message.response.sources} onNavigate={onNavigate} />
            {message.response.permissionNotes.map((note) => (
              <p className="mt-3 text-xs text-cocoa-500" key={note}>{note}</p>
            ))}
            {message.response.warnings.map((warning) => (
              <p className="mt-2 text-xs text-amber-800" key={warning}>{warning}</p>
            ))}
          </>
        ) : null}
      </div>
    </article>
  );
}
