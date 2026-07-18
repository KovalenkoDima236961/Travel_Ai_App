import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderOptions } from "@testing-library/react";
import { NextIntlClientProvider } from "next-intl";
import type { ReactElement, ReactNode } from "react";
import type { Mock } from "vitest";
import type { AuthUser } from "@/shared/api/auth";
import en from "../messages/en.json";
import es from "../messages/es.json";
import fr from "../messages/fr.json";
import uk from "../messages/uk.json";

const messages = { en, es, fr, uk };
type TestLocale = keyof typeof messages;

export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { gcTime: Infinity, retry: false, staleTime: Infinity }
    }
  });
}

type RenderWithProvidersOptions = Omit<RenderOptions, "wrapper"> & {
  locale?: TestLocale;
  queryClient?: QueryClient;
};

export function renderWithProviders(
  ui: ReactElement,
  { locale = "en", queryClient = createTestQueryClient(), ...options }: RenderWithProvidersOptions = {}
) {
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <NextIntlClientProvider locale={locale} messages={messages[locale]} timeZone="Europe/Bratislava">
          {children}
        </NextIntlClientProvider>
      </QueryClientProvider>
    );
  }

  return { queryClient, ...render(ui, { wrapper: Wrapper, ...options }) };
}

export function createMockAuthState(overrides: Partial<MockAuthState> = {}): MockAuthState {
  const user = overrides.user === undefined ? ownerUser : overrides.user;
  return {
    user,
    isAuthenticated: Boolean(user),
    isLoading: false,
    login: async () => undefined,
    logout: async () => undefined,
    refresh: async () => ({ accessToken: "test-access-token", refreshToken: "test-refresh-token" }),
    register: async () => undefined,
    ...overrides
  };
}

export type MockAuthState = {
  user: AuthUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (credentials: { email: string; password: string }) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<{ accessToken: string; refreshToken: string }>;
  register: (credentials: { email: string; password: string }) => Promise<void>;
};

const ownerUser: AuthUser = {
  id: "10000000-0000-4000-8000-000000000001",
  email: "owner@example.test",
  createdAt: "2026-01-15T09:00:00Z"
};

export const navigationMocks = (
  globalThis as typeof globalThis & {
    __TRAVEL_AI_NAVIGATION_MOCKS__: {
      pathname: string;
      searchParams: URLSearchParams;
      router: {
        back: Mock;
        forward: Mock;
        prefetch: Mock;
        push: Mock;
        refresh: Mock;
        replace: Mock;
      };
    };
  }
).__TRAVEL_AI_NAVIGATION_MOCKS__;
export * from "@testing-library/react";
export { default as userEvent } from "@testing-library/user-event";
