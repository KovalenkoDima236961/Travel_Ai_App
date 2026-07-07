"use client";

import { FormEvent, useState } from "react";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";

type PublicShareUnlockFormProps = {
  onUnlock(password: string): Promise<void>;
  loading?: boolean;
  error?: string | null;
};

export function PublicShareUnlockForm({
  onUnlock,
  loading = false,
  error
}: PublicShareUnlockFormProps) {
  const [password, setPassword] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await onUnlock(password);
  }

  return (
    <div className="mx-auto w-full max-w-md px-4 py-12 sm:px-6 lg:px-8">
      <Card>
        <form className="space-y-4" onSubmit={submit}>
          <div>
            <h1 className="text-xl font-semibold text-slate-950">
              This shared trip is password protected
            </h1>
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700" htmlFor="share-password">
              Password
            </label>
            <Input
              autoComplete="current-password"
              disabled={loading}
              id="share-password"
              onChange={(event) => setPassword(event.target.value)}
              type="password"
              value={password}
            />
          </div>
          {error ? <p className="text-sm font-medium text-red-700">{error}</p> : null}
          <Button className="w-full" disabled={loading || !password} type="submit">
            {loading ? "Unlocking..." : "Unlock"}
          </Button>
        </form>
      </Card>
    </div>
  );
}
