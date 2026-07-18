// @vitest-environment jsdom

import { describe, expect, it, vi } from "vitest";
import { CreateTripForm } from "@/_pages/new-trip/ui/CreateTripForm";
import { createMockAuthState, renderWithProviders, screen, userEvent } from "../../../../test/test-utils";

const mocks = vi.hoisted(() => ({
  auth: null as ReturnType<typeof createMockAuthState> | null,
  markFirstTripCreated: vi.fn()
}));

vi.mock("@/components/auth/AuthProvider", () => ({
  useAuth: () => mocks.auth
}));

vi.mock("@/components/workspaces/WorkspaceProvider", () => ({
  useWorkspaces: () => ({
    currentScope: "personal",
    currentWorkspace: null,
    currentWorkspaceId: null,
    editableWorkspaces: [],
    workspaces: []
  })
}));

vi.mock("@/hooks/useOnboardingState", () => ({
  useOnboardingState: () => ({ markFirstTripCreated: mocks.markFirstTripCreated })
}));

vi.mock("@/hooks/usePersonalization", () => ({
  usePreferenceCompleteness: () => ({ data: null })
}));

vi.mock("@/hooks/usePlanningConstraintsPreview", () => ({
  usePlanningConstraintsPreview: () => ({
    data: null,
    error: null,
    isError: false,
    isPending: false,
    mutate: vi.fn()
  })
}));

describe("CreateTripForm", () => {
  it("keeps the user on the essentials step until required fields are valid", async () => {
    mocks.auth = createMockAuthState();
    const user = userEvent.setup();
    renderWithProviders(<CreateTripForm />);

    await user.click(screen.getByRole("button", { name: "Continue" }));

    expect(await screen.findByText("Add at least one destination or route stop.")).toBeVisible();
    expect(screen.getByText("Choose a start date.")).toBeVisible();
    expect(screen.getByRole("heading", { name: "Where and when?" })).toBeVisible();

    await user.type(screen.getByPlaceholderText("City, region, or country"), "Vienna");
    await user.type(document.querySelector<HTMLInputElement>("#startDate")!, "2026-04-10");
    await user.click(screen.getByRole("button", { name: "Continue" }));

    expect(await screen.findByRole("heading", { name: "Who is going?" })).toBeVisible();
  });
});
