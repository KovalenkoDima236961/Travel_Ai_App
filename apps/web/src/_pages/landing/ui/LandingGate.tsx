"use client";

import { ReactNode, useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/auth/AuthProvider";

type LandingGateProps = {
  children: ReactNode;
};

/**
 * The landing page is for logged-out visitors. Auth lives in localStorage (not a
 * cookie), so this can't be decided in middleware — we redirect on the client once
 * AuthProvider has confirmed the session. On first paint the user is unresolved, so
 * the marketing page renders (matching SSR and keeping hydration stable) and is
 * replaced only if a signed-in session is found.
 */
export function LandingGate({ children }: LandingGateProps) {
  const router = useRouter();
  const { isAuthenticated } = useAuth();

  useEffect(() => {
    if (isAuthenticated) {
      router.replace("/trips");
    }
  }, [isAuthenticated, router]);

  if (isAuthenticated) {
    return null;
  }

  return <>{children}</>;
}
