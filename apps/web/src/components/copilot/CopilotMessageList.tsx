import { CopilotMessageBubble } from "./CopilotMessageBubble";
import type { CopilotMessage } from "@/types/copilot";

export function CopilotMessageList({ messages, onNavigate }: { messages: CopilotMessage[]; onNavigate?: () => void }) {
  if (messages.length === 0) {
    return null;
  }
  return (
    <div aria-live="polite" className="space-y-3">
      {messages.map((message) => <CopilotMessageBubble key={message.id} message={message} onNavigate={onNavigate} />)}
    </div>
  );
}
