"use client";

import { useMutation } from "@tanstack/react-query";
import { previewCalendarAvailabilityImport } from "@/lib/api/calendar-free-busy";
import type { CalendarImportPreviewRequest } from "@/types/calendar-free-busy";

export function useCalendarAvailabilityImportPreview(tripId: string) {
  return useMutation({
    mutationFn: (input: CalendarImportPreviewRequest) =>
      previewCalendarAvailabilityImport(tripId, input)
  });
}
