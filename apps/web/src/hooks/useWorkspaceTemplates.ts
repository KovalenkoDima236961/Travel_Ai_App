"use client";

import { useQuery } from "@tanstack/react-query";
import {
  listWorkspaceTripTemplates,
  tripTemplateKeys
} from "@/lib/api/trip-templates";
import type { ListTripTemplatesParams } from "@/types/trip-template";

export function useWorkspaceTemplates(
  workspaceId: string,
  params: Omit<ListTripTemplatesParams, "workspaceId" | "visibility"> = {}
) {
  return useQuery({
    queryKey: tripTemplateKeys.workspace(workspaceId, params),
    queryFn: () => listWorkspaceTripTemplates(workspaceId, params),
    enabled: Boolean(workspaceId)
  });
}
