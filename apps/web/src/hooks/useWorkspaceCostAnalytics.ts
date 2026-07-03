import { useQuery } from "@tanstack/react-query";
import {
  costAnalyticsKeys,
  getWorkspaceCostAnalytics
} from "@/lib/api/cost-analytics";
import type { WorkspaceCostAnalyticsParams } from "@/types/cost-analytics";

export function useWorkspaceCostAnalytics({
  workspaceId,
  params,
  enabled = true
}: {
  workspaceId: string;
  params?: WorkspaceCostAnalyticsParams;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: costAnalyticsKeys.workspace(workspaceId, params),
    queryFn: () => getWorkspaceCostAnalytics(workspaceId, params),
    enabled: enabled && Boolean(workspaceId)
  });
}
