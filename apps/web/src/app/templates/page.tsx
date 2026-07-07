import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { TemplatesPageContent } from "@/_pages/templates/ui/TemplatesPageContent";

export default function TemplatesPage() {
  return (
    <ProtectedRoute>
      <TemplatesPageContent />
    </ProtectedRoute>
  );
}
