"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { usePathname } from "next/navigation";
import { ReactNode, useEffect, useState } from "react";
import { AuthProvider, useAuth } from "@/components/auth/AuthProvider";
import { useOfflineSync } from "@/hooks/useOfflineSync";
import { registerServiceWorker } from "@/lib/push/register-service-worker";

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

  useEffect(() => {
    registerServiceWorker().catch(() => {
      // Offline app shell support is best-effort and should not block the app.
    });
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <OfflineSyncController />
        {children}
      </AuthProvider>
    </QueryClientProvider>
  );
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
