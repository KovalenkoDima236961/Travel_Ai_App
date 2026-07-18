// @vitest-environment jsdom

import { axe } from "jest-axe";
import { describe, expect, it, vi } from "vitest";
import { LoginForm } from "@/_pages/auth/ui/LoginForm";
import { RegisterForm } from "@/_pages/auth/ui/RegisterForm";
import {
  createMockAuthState,
  navigationMocks,
  renderWithProviders,
  screen,
  userEvent
} from "../../../../test/test-utils";

const authState = vi.hoisted(() => ({ current: null as ReturnType<typeof createMockAuthState> | null }));

vi.mock("@/components/auth/AuthProvider", () => ({
  useAuth: () => authState.current
}));

describe("auth forms", () => {
  it("shows behavioral validation errors without calling login", async () => {
    const user = userEvent.setup();
    authState.current = createMockAuthState({ login: vi.fn(async () => undefined) });
    renderWithProviders(<LoginForm />);

    await user.click(screen.getByRole("button", { name: "Log in" }));

    expect(await screen.findByText("Enter a valid email address")).toBeVisible();
    expect(screen.getByText("Password is required")).toBeVisible();
    expect(authState.current.login).not.toHaveBeenCalled();
  });

  it("normalizes credentials, submits once, and routes to the safe next page", async () => {
    const login = vi.fn(async () => undefined);
    authState.current = createMockAuthState({ login });
    navigationMocks.searchParams = new URLSearchParams("next=/settings");
    const user = userEvent.setup();
    renderWithProviders(<LoginForm />);

    await user.type(screen.getByLabelText("Email"), "  OWNER@EXAMPLE.TEST  ");
    await user.type(screen.getByLabelText("Password"), "TestPassword1");
    await user.click(screen.getByRole("button", { name: "Log in" }));

    expect(login).toHaveBeenCalledWith({ email: "owner@example.test", password: "TestPassword1" });
    expect(navigationMocks.router.push).toHaveBeenCalledWith("/settings");
  });

  it("rejects mismatched registration passwords and has no obvious axe violations", async () => {
    const register = vi.fn(async () => undefined);
    authState.current = createMockAuthState({ register });
    const user = userEvent.setup();
    const { container } = renderWithProviders(<RegisterForm />);

    await user.type(screen.getByLabelText("Email"), "owner@example.test");
    await user.type(screen.getByLabelText("Password"), "TestPassword1");
    await user.type(screen.getByLabelText("Confirm password"), "Different1");
    await user.click(screen.getByRole("button", { name: "Create account" }));

    expect(await screen.findByText("Passwords must match")).toBeVisible();
    expect(register).not.toHaveBeenCalled();
    expect(await axe(container)).toHaveNoViolations();
  });
});
