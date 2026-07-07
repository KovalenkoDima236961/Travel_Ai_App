import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { TemplateDetailPageContent } from "@/_pages/template-detail/ui/TemplateDetailPageContent";

export default function TemplateDetailPage() {
  return (
    <ProtectedRoute>
      <TemplateDetailPageContent />
    </ProtectedRoute>
  );
}
