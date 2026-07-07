"use client";

import { PageContainer } from "@/components/layout/PageContainer";
import { NewWorkspaceForm } from "./NewWorkspaceForm";

export function NewWorkspacePageContent() {
  return (
    <PageContainer className="max-w-3xl">
      <NewWorkspaceForm />
    </PageContainer>
  );
}
