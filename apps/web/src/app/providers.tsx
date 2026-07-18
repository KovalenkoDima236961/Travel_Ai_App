"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { usePathname } from "next/navigation";
import { ReactNode, useEffect, useState } from "react";
import { AuthProvider, useAuth } from "@/components/auth/AuthProvider";
import { GlobalCommandPalette } from "@/components/command-palette/GlobalCommandPalette";
import { I18nProvider } from "@/components/i18n/I18nProvider";
import { AppUpdateBanner } from "@/components/pwa/AppUpdateBanner";
import { PwaInstallPrompt } from "@/components/pwa/PwaInstallPrompt";
import { WorkspaceProvider } from "@/components/workspaces/WorkspaceProvider";
import { useOfflineSync } from "@/hooks/useOfflineSync";
import { registerServiceWorker } from "@/lib/push/register-service-worker";
import { FeatureFlagProvider, FeatureGate, useFeatureFlag } from "@/lib/feature-flags/FeatureFlagProvider";

type ProvidersProps = {
  children: ReactNode;
};

export function Providers({ children }: ProvidersProps) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            refetchOnWindowFocus: false,
            retry: 1,
            staleTime: 30_000
          }
        }
      })
  );

  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <I18nProvider>
          <FeatureFlagProvider>
            <WorkspaceProvider>
              <RuntimeFeatureControllers />
              <OfflineSyncController />
              <GlobalCommandPalette />
              <AppUpdateBanner />
              <FeatureGate flag="offline_mode_enabled">
                <PwaInstallPrompt />
              </FeatureGate>
              {children}
            </WorkspaceProvider>
          </FeatureFlagProvider>
        </I18nProvider>
      </AuthProvider>
    </QueryClientProvider>
  );
}

function RuntimeFeatureControllers() {
  const offlineModeEnabled = useFeatureFlag("offline_mode_enabled");

  useEffect(() => {
    if (!offlineModeEnabled) {
      return;
    }
    registerServiceWorker().catch(() => {
      // Offline app shell support is best-effort and should not block the app.
    });
  }, [offlineModeEnabled]);

  return null;
}

function OfflineSyncController() {
  const { user, isLoading } = useAuth();
  const pathname = usePathname();
  const isTripDetailPage = /^\/trips\/[^/]+/.test(pathname ?? "");

  useOfflineSync({
    userId: user?.id,
    enabled: Boolean(user?.id) && !isLoading && !isTripDetailPage
  });

  return null;
}
