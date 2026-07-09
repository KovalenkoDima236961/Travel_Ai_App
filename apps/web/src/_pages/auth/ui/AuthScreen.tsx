"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/shared/lib/cn";
import { instrumentSans, newsreader } from "./fonts";
import { GlobeIcon } from "./icons";
import { LoginForm } from "./LoginForm";
import { RegisterForm } from "./RegisterForm";

const TAB_BASE = "inline-flex h-[34px] items-center rounded-full px-5 text-[13.5px] transition";
const TAB_ACTIVE = "bg-cocoa-900 font-semibold text-sand-150";
const TAB_IDLE = "font-medium text-cocoa-500 hover:text-cocoa-900";

/**
 * Split-screen auth (Auth.dc.html). The active tab is derived from the route —
 * `/register` shows registration, everything else shows login — so the tabs are
 * real links: deep-links, the header's Log in / Get started, and the post-logout
 * redirect all land on the correct panel without any local mode state to drift.
 */
export function AuthScreen() {
  const pathname = usePathname();
  const isRegister = pathname === "/register";

  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "grid min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC] lg:grid-cols-2"
      )}
    >
      {/* Brand + photo panel — desktop only. */}
      <section className="relative hidden overflow-hidden bg-cocoa-900 lg:block">
        {/* Photo slot: on-brand gradient placeholder; swap for a real travel photo. */}
        <div
          className="absolute inset-0 opacity-[0.55]"
          style={{
            backgroundImage:
              "radial-gradient(120% 90% at 70% 12%, #F7CDA1 0%, transparent 50%), linear-gradient(165deg, #D98A5A 0%, #B5613C 45%, #221A14 100%)"
          }}
        />
        <div className="pointer-events-none absolute inset-0 bg-gradient-to-b from-cocoa-900/35 to-cocoa-900/75" />
        <div className="pointer-events-none relative flex h-full flex-col justify-between p-12">
          <div className="flex items-center gap-2.5 text-sand-150">
            <span className="flex h-[34px] w-[34px] items-center justify-center rounded-full bg-clay text-sand-100">
              <GlobeIcon className="h-[19px] w-[19px]" />
            </span>
            <span className="font-newsreader text-[20px] font-semibold">Travel AI Planner</span>
          </div>
          <div>
            <p className="max-w-[420px] font-newsreader text-[34px] font-medium leading-[1.2] tracking-[-0.015em] text-sand-150">
              Every great trip starts with a <em className="text-clay-glow">single idea.</em>
            </p>
            <p className="mt-4 max-w-[380px] text-[15px] text-sand-150/70">
              Plan smarter with AI-drafted itineraries you can shape into your own.
            </p>
          </div>
        </div>
      </section>

      {/* Form panel. */}
      <section className="flex items-center justify-center p-6 sm:p-10 lg:p-12">
        <div className="w-full max-w-[400px]">
          <div className="inline-flex gap-1 rounded-full border border-[#E8DFD3] bg-white p-1">
            <Link
              href="/login"
              aria-current={!isRegister ? "page" : undefined}
              className={cn(TAB_BASE, isRegister ? TAB_IDLE : TAB_ACTIVE)}
            >
              Log in
            </Link>
            <Link
              href="/register"
              aria-current={isRegister ? "page" : undefined}
              className={cn(TAB_BASE, isRegister ? TAB_ACTIVE : TAB_IDLE)}
            >
              Register
            </Link>
          </div>

          <h1 className="mt-7 font-newsreader text-[34px] font-medium tracking-[-0.02em] text-cocoa-900">
            {isRegister ? "Create your account" : "Welcome back"}
          </h1>
          <p className="mt-2.5 text-[14.5px] text-cocoa-400">
            {isRegister
              ? "Start planning your next trip in minutes."
              : "Log in to pick up where you left off."}
          </p>

          {isRegister ? <RegisterForm /> : <LoginForm />}

          <p className="mt-6 text-[14px] text-cocoa-400">
            {isRegister ? "Already registered?" : "No account yet?"}{" "}
            <Link
              href={isRegister ? "/login" : "/register"}
              className="font-semibold text-clay-deep transition hover:text-clay"
            >
              {isRegister ? "Log in" : "Register"}
            </Link>
          </p>
        </div>
      </section>
    </div>
  );
}
