import { Button } from "@/shared/ui/button";

type CalendarConnectionPromptProps = {
  isPending: boolean;
  onConnect: () => void;
};

export function CalendarConnectionPrompt({
  isPending,
  onConnect
}: CalendarConnectionPromptProps) {
  return (
    <div className="rounded-md border border-slate-200 bg-slate-50 p-4">
      <h3 className="text-sm font-semibold text-slate-950">Calendar is not connected</h3>
      <p className="mt-2 text-sm leading-6 text-slate-600">
        Connect Google Calendar before importing free/busy blocks.
      </p>
      <Button className="mt-4" disabled={isPending} onClick={onConnect} size="sm" type="button">
        {isPending ? "Connecting..." : "Connect Google Calendar"}
      </Button>
    </div>
  );
}
