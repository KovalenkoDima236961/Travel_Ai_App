"use client";

import { FormEvent, useState } from "react";
import { LockClosedIcon } from "./icons";

type PublicShareUnlockProps = {
  onUnlock(password: string): Promise<void>;
  loading?: boolean;
  error?: string | null;
};

/**
 * Warm slice-local unlock screen for password-protected shares. The unlock flow
 * (submit → sessionStorage token) lives in PublicSharePageContent; this component
 * only presents the form in the shared screen's palette.
 */
export function PublicShareUnlock({ onUnlock, loading = false, error }: PublicShareUnlockProps) {
  const [password, setPassword] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await onUnlock(password);
  }

  return (
    <div className="mx-auto flex max-w-[440px] flex-col items-center px-6 py-16 sm:py-24">
      <span className="flex h-14 w-14 items-center justify-center rounded-full bg-clay-tint text-clay-deep">
        <LockClosedIcon className="h-6 w-6" />
      </span>
      <h1 className="mt-6 text-center font-newsreader text-[28px] font-medium leading-tight tracking-[-0.01em] text-cocoa-900">
        This shared trip is password protected
      </h1>
      <p className="mt-2.5 text-center text-[14.5px] leading-[1.6] text-cocoa-500">
        Enter the password to view the itinerary.
      </p>
      <form className="mt-7 w-full" onSubmit={submit}>
        <label className="block text-[13px] font-medium text-cocoa-700" htmlFor="share-password">
          Password
        </label>
        <input
          autoComplete="current-password"
          disabled={loading}
          id="share-password"
          onChange={(event) => setPassword(event.target.value)}
          type="password"
          value={password}
          className="mt-2 h-11 w-full rounded-[12px] border border-sand-400 bg-white px-4 text-[14.5px] text-cocoa-900 outline-none transition placeholder:text-[#B09E8A] focus:border-clay focus:ring-2 focus:ring-clay/15 disabled:opacity-60"
        />
        {error ? (
          <p className="mt-3 text-[13.5px] font-medium text-[#B3402E]">{error}</p>
        ) : null}
        <button
          type="submit"
          disabled={loading || !password}
          className="mt-5 inline-flex h-11 w-full items-center justify-center rounded-full bg-clay text-[14.5px] font-semibold text-sand-100 transition hover:bg-clay-dark disabled:cursor-not-allowed disabled:opacity-60"
        >
          {loading ? "Unlocking…" : "Unlock"}
        </button>
      </form>
    </div>
  );
}
