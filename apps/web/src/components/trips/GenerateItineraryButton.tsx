"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/Button";
import { generateItinerary, tripKeys } from "@/lib/api/trips";
import { getErrorMessage } from "@/lib/utils";

type GenerateItineraryButtonProps = {
  tripId: string;
};

export function GenerateItineraryButton({ tripId }: GenerateItineraryButtonProps) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: () => generateItinerary(tripId),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.lists() })
      ]);
    }
  });

  return (
    <div className="flex flex-col items-start gap-2 sm:items-end">
      <Button disabled={mutation.isPending} onClick={() => mutation.mutate()}>
        {mutation.isPending ? "Generating..." : "Generate itinerary"}
      </Button>
      {mutation.isError ? (
        <p className="max-w-xs text-sm text-red-700" role="alert">
          {getErrorMessage(mutation.error, "Could not generate itinerary.")}
        </p>
      ) : null}
    </div>
  );
}
