import { Suspense } from "react";
import { AuthScreen } from "@/_pages/auth/ui/AuthScreen";

export default function RegisterPage() {
  return (
    <Suspense fallback={<div className="min-h-screen bg-sand-50" />}>
      <AuthScreen />
    </Suspense>
  );
}
