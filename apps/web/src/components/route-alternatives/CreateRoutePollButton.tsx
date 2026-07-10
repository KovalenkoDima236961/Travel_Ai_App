"use client";

import { useState } from "react";
import { useCreateRouteAlternativesPoll } from "@/hooks/useCreateRouteAlternativesPoll";
import { getErrorMessage } from "@/lib/utils";
import type { RouteAlternativeSession } from "@/types/route-alternatives";

type CreateRoutePollButtonProps = {
  tripId: string;
  session: RouteAlternativeSession | null;
  disabled?: boolean;
};

export function CreateRoutePollButton({ tripId, session, disabled = false }: CreateRoutePollButtonProps) {
  const [created, setCreated] = useState(false);
  const mutation = useCreateRouteAlternativesPoll(tripId, session?.id);

  if (!session || session.alternatives.length === 0) {
    return null;
  }

  async function createPoll() {
    if (!session) {
      return;
    }
    await mutation.mutateAsync({
      title: "Which route should we choose?",
      alternativeIds: session.alternatives.map((alternative) => alternative.id)
    });
    setCreated(true);
  }

  return (
    <div className="flex flex-wrap items-center gap-3">
      <button
        type="button"
        disabled={disabled || mutation.isPending || created}
        onClick={createPoll}
        className="h-10 rounded-full border border-clay bg-clay-tint px-4 text-[13px] font-semibold text-clay-deep transition hover:bg-clay hover:text-sand-100 disabled:cursor-not-allowed disabled:opacity-60"
      >
        {created ? "Poll created" : mutation.isPending ? "Creating poll..." : "Create poll from routes"}
      </button>
      {mutation.isError ? (
        <p className="text-[12px] text-red-700">
          {getErrorMessage(mutation.error, "Could not create route poll.")}
        </p>
      ) : null}
    </div>
  );
}
