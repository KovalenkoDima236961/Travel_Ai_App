import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { GettingStartedPageContent } from "@/_pages/getting-started/ui/GettingStartedPageContent";

export default function GettingStartedPage() {
  return <ProtectedRoute><GettingStartedPageContent /></ProtectedRoute>;
}
