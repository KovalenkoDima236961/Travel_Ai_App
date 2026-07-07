"use client";

import { useEffect, useState } from "react";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import type {
  CreateTripTravelerInput,
  TripTraveler,
  TripTravelerRole
} from "@/entities/cost-splitting/model";

type TravelerFormDialogProps = {
  open: boolean;
  traveler?: TripTraveler | null;
  isSaving?: boolean;
  error?: string | null;
  onClose: () => void;
  onSubmit: (input: CreateTripTravelerInput) => void;
};

export function TravelerFormDialog({
  open,
  traveler,
  isSaving = false,
  error,
  onClose,
  onSubmit
}: TravelerFormDialogProps) {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<TripTravelerRole>("traveler");
  const [localError, setLocalError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    setName(traveler?.name ?? "");
    setEmail(traveler?.email ?? "");
    setRole(traveler?.role ?? "traveler");
    setLocalError(null);
  }, [open, traveler]);

  if (!open) {
    return null;
  }

  function submit() {
    const trimmedName = name.trim();
    const trimmedEmail = email.trim();
    if (!trimmedName) {
      setLocalError("Name is required.");
      return;
    }
    if (trimmedEmail && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmedEmail)) {
      setLocalError("Enter a valid email address.");
      return;
    }
    setLocalError(null);
    onSubmit({
      name: trimmedName,
      email: trimmedEmail || null,
      role
    });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-950/35 px-4 py-8">
      <div className="w-full max-w-lg rounded-lg border border-slate-200 bg-white p-5 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <h2 className="text-lg font-semibold text-slate-950">
            {traveler ? "Edit traveler" : "Add traveler"}
          </h2>
          <Button onClick={onClose} size="sm" type="button" variant="ghost">
            Close
          </Button>
        </div>

        <div className="mt-5 space-y-4">
          <label className="block text-sm font-medium text-slate-700">
            Name
            <Input className="mt-1" onChange={(event) => setName(event.target.value)} value={name} />
          </label>
          <label className="block text-sm font-medium text-slate-700">
            Email
            <Input
              className="mt-1"
              onChange={(event) => setEmail(event.target.value)}
              type="email"
              value={email}
            />
          </label>
          <label className="block text-sm font-medium text-slate-700">
            Role
            <Select
              className="mt-1"
              onChange={(event) => setRole(event.target.value as TripTravelerRole)}
              value={role}
            >
              <option value="traveler">Traveler</option>
              <option value="organizer">Organizer</option>
            </Select>
          </label>
        </div>

        {localError || error ? (
          <p className="mt-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
            {localError ?? error}
          </p>
        ) : null}

        <div className="mt-6 flex justify-end gap-2">
          <Button disabled={isSaving} onClick={onClose} type="button" variant="secondary">
            Cancel
          </Button>
          <Button disabled={isSaving} onClick={submit} type="button">
            {isSaving ? "Saving..." : "Save"}
          </Button>
        </div>
      </div>
    </div>
  );
}
