"use client";

import { createContext, type ReactNode, useContext, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { getPublicFeatureFlags, featureFlagKeys } from "./client";
import { fallbackFeatureFlags, type FeatureFlagKey } from "./types";

type FeatureFlagContextValue = {
  flags: Record<FeatureFlagKey, boolean>;
  isLoading: boolean;
  hasLoadError: boolean;
};

const FeatureFlagContext = createContext<FeatureFlagContextValue | null>(null);

export function FeatureFlagProvider({ children }: { children: ReactNode }) {
  const query = useQuery({
    queryKey: featureFlagKeys.public,
    queryFn: getPublicFeatureFlags,
    staleTime: 30_000,
    refetchOnWindowFocus: true,
    retry: 1
  });
  const value = useMemo<FeatureFlagContextValue>(() => ({
    flags: { ...fallbackFeatureFlags(), ...query.data?.flags },
    isLoading: query.isLoading,
    hasLoadError: query.isError
  }), [query.data?.flags, query.isError, query.isLoading]);
  return <FeatureFlagContext.Provider value={value}>{children}</FeatureFlagContext.Provider>;
}

export function useFeatureFlags() {
  return useContext(FeatureFlagContext) ?? {
    flags: fallbackFeatureFlags(), isLoading: true, hasLoadError: false
  };
}

export function useFeatureFlag(key: FeatureFlagKey): boolean {
  return useFeatureFlags().flags[key] === true;
}

export function FeatureGate({ flag, children, fallback = null }: {
  flag: FeatureFlagKey;
  children: ReactNode;
  fallback?: ReactNode;
}) {
  return useFeatureFlag(flag) ? <>{children}</> : <>{fallback}</>;
}
