import "@testing-library/jest-dom/vitest";
import "fake-indexeddb/auto";
import { cleanup } from "@testing-library/react";
import { afterAll, afterEach, beforeAll, expect, vi } from "vitest";
import { toHaveNoViolations } from "jest-axe";
import { server } from "./msw/server";

expect.extend(toHaveNoViolations);

const navigationMocks = vi.hoisted(() => ({
  pathname: "/",
  searchParams: new URLSearchParams(),
  router: {
    back: vi.fn(),
    forward: vi.fn(),
    prefetch: vi.fn(),
    push: vi.fn(),
    refresh: vi.fn(),
    replace: vi.fn()
  }
}));

Object.defineProperty(globalThis, "__TRAVEL_AI_NAVIGATION_MOCKS__", {
  configurable: true,
  value: navigationMocks
});

vi.mock("next/navigation", () => ({
  useParams: () => ({}),
  usePathname: () => navigationMocks.pathname,
  useRouter: () => navigationMocks.router,
  useSearchParams: () => navigationMocks.searchParams
}));

vi.mock("next/font/google", () => ({
  Instrument_Sans: () => ({ className: "font-instrument", variable: "font-instrument" }),
  Newsreader: () => ({ className: "font-newsreader", variable: "font-newsreader" })
}));

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));

afterEach(() => {
  cleanup();
  server.resetHandlers();
  if (typeof window !== "undefined") {
    window.localStorage?.clear?.();
    window.sessionStorage?.clear?.();
  }
  navigationMocks.pathname = "/";
  navigationMocks.searchParams = new URLSearchParams();
});

afterAll(() => server.close());

class ObserverMock {
  disconnect() {}
  observe() {}
  takeRecords() {
    return [];
  }
  unobserve() {}
}

class BroadcastChannelMock {
  name: string;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onmessageerror: ((event: MessageEvent) => void) | null = null;

  constructor(name: string) {
    this.name = name;
  }

  addEventListener() {}
  close() {}
  dispatchEvent() {
    return true;
  }
  postMessage() {}
  removeEventListener() {}
}

class NotificationMock {
  static permission: NotificationPermission = "default";
  static requestPermission = vi.fn(async () => "granted" as NotificationPermission);

  close = vi.fn();

  constructor(_title: string, _options?: NotificationOptions) {}
}

Object.defineProperties(globalThis, {
  BroadcastChannel: { configurable: true, writable: true, value: BroadcastChannelMock },
  IntersectionObserver: { configurable: true, writable: true, value: ObserverMock },
  Notification: { configurable: true, writable: true, value: NotificationMock },
  PushManager: { configurable: true, writable: true, value: class PushManagerMock {} },
  ResizeObserver: { configurable: true, writable: true, value: ObserverMock }
});

if (typeof window !== "undefined" && typeof navigator !== "undefined") {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: vi.fn((query: string): MediaQueryList => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      addListener: vi.fn(),
      dispatchEvent: vi.fn(() => true),
      removeEventListener: vi.fn(),
      removeListener: vi.fn()
    }))
  });

  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: {
      readText: vi.fn(async () => ""),
      writeText: vi.fn(async () => undefined)
    }
  });

  Object.defineProperty(navigator, "serviceWorker", {
    configurable: true,
    value: {
      addEventListener: vi.fn(),
      controller: null,
      getRegistration: vi.fn(async () => undefined),
      getRegistrations: vi.fn(async () => []),
      ready: Promise.resolve({}),
      register: vi.fn(async () => ({
        active: null,
        installing: null,
        pushManager: {
          getSubscription: vi.fn(async () => null),
          subscribe: vi.fn(async () => null)
        },
        unregister: vi.fn(async () => true),
        waiting: null
      })),
      removeEventListener: vi.fn()
    }
  });
}
