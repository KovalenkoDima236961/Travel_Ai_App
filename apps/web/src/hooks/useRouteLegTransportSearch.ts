"use client";

import { useMutation } from "@tanstack/react-query";
import {
  searchRouteLegTransportOptions,
  transportKeys
} from "@/lib/api/transport";
import type {
  SearchRouteLegTransportInput,
  TransportSearchResponse
} from "@/types/transport";

export function useRouteLegTransportSearch(tripId: string, legId: string) {
  return useMutation<TransportSearchResponse, Error, SearchRouteLegTransportInput | undefined>({
    mutationKey: transportKeys.routeLeg(tripId, legId),
    mutationFn: (input) => searchRouteLegTransportOptions(tripId, legId, input ?? {})
  });
}
