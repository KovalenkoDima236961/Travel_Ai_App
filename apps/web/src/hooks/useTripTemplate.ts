"use client";

import { useQuery } from "@tanstack/react-query";
import { getTripTemplate, tripTemplateKeys } from "@/lib/api/trip-templates";

export function useTripTemplate(templateId: string) {
  return useQuery({
    queryKey: tripTemplateKeys.detail(templateId),
    queryFn: () => getTripTemplate(templateId),
    enabled: Boolean(templateId)
  });
}
