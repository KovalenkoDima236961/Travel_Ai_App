import { Suspense } from "react";
import { PageContainer } from "@/components/layout/PageContainer";
import { LoginPageContent } from "@/_pages/auth-login/ui/LoginPageContent";

export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <PageContainer className="max-w-lg">
          <div className="text-sm text-slate-600">Loading login...</div>
        </PageContainer>
      }
    >
      <LoginPageContent />
    </Suspense>
  );
}
