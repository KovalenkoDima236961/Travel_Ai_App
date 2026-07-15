import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  evaluateTripPolicy,
  getTripPolicyEvaluation,
  workspacePolicyKeys
} from "@/lib/api/workspace-policies";
import { tripHealthKeys } from "@/lib/api/trip-health";

export function useTripPolicyEvaluation(tripId: string, enabled = true) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: workspacePolicyKeys.evaluation(tripId),
    queryFn: () => getTripPolicyEvaluation(tripId),
    enabled: enabled && Boolean(tripId)
  });
  const evaluate = useMutation({
    mutationFn: () => evaluateTripPolicy(tripId),
    onSuccess: (result) => {
      queryClient.setQueryData(workspacePolicyKeys.evaluation(tripId), result);
      void queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) });
    }
  });
  return { query, evaluate };
}
