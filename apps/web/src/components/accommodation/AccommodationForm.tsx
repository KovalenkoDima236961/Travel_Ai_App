"use client";

import { useState } from "react";
import { AttachPlaceDialog } from "@/components/places/AttachPlaceDialog";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import {
  ACCOMMODATION_TYPES,
  type AccommodationType,
  type TripAccommodation
} from "@/types/accommodation";
import type { Place } from "@/types/place";

type AccommodationFormProps = {
  initial?: TripAccommodation | null;
  defaultCurrency: string;
  destination?: string;
  isSaving?: boolean;
  onSave: (accommodation: TripAccommodation) => void;
  onCancel: () => void;
  onClear?: () => void;
};

export function AccommodationForm({
  initial,
  defaultCurrency,
  destination,
  isSaving = false,
  onSave,
  onCancel,
  onClear
}: AccommodationFormProps) {
  const [name, setName] = useState(initial?.name ?? "");
  const [type, setType] = useState<AccommodationType>(initial?.type ?? "hotel");
  const [address, setAddress] = useState(initial?.address ?? "");
  const [checkInDate, setCheckInDate] = useState(initial?.checkInDate ?? "");
  const [checkOutDate, setCheckOutDate] = useState(initial?.checkOutDate ?? "");
  const [amount, setAmount] = useState(
    initial?.estimatedCost?.amount != null ? String(initial.estimatedCost.amount) : ""
  );
  const [currency, setCurrency] = useState(
    (initial?.estimatedCost?.currency ?? defaultCurrency ?? "EUR").toUpperCase()
  );
  const [notes, setNotes] = useState(initial?.notes ?? "");
  const [place, setPlace] = useState<Place | null>(initial?.place ?? null);
  const [isPlaceDialogOpen, setIsPlaceDialogOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function handleSave() {
    const trimmedName = name.trim();
    const trimmedAddress = address.trim();
    const trimmedNotes = notes.trim();
    if (!trimmedName) {
      setError("Enter an accommodation name.");
      return;
    }
    if (trimmedName.length > 200) {
      setError("Accommodation name must be 200 characters or fewer.");
      return;
    }
    if (trimmedAddress.length > 500) {
      setError("Address must be 500 characters or fewer.");
      return;
    }
    if (trimmedNotes.length > 1000) {
      setError("Notes must be 1000 characters or fewer.");
      return;
    }
    if (checkInDate && checkOutDate && checkOutDate <= checkInDate) {
      setError("Check-out date must be after check-in date.");
      return;
    }

    const normalizedCurrency = currency.trim().toUpperCase();
    const trimmedAmount = amount.trim();
    let parsedAmount: number | null = null;
    if (trimmedAmount) {
      const nextAmount = Number(trimmedAmount);
      if (Number.isNaN(nextAmount)) {
        setError("Enter a valid accommodation cost.");
        return;
      }
      parsedAmount = nextAmount;
    }
    if (parsedAmount != null && parsedAmount < 0) {
      setError("Accommodation cost must be 0 or more.");
      return;
    }
    if (parsedAmount != null && !/^[A-Z]{3}$/.test(normalizedCurrency)) {
      setError("Currency must be a 3-letter code (e.g. EUR).");
      return;
    }

    setError(null);
    onSave({
      name: trimmedName,
      type,
      address: trimmedAddress || null,
      place,
      checkInDate: checkInDate || null,
      checkOutDate: checkOutDate || null,
      estimatedCost:
        parsedAmount != null
          ? {
              amount: parsedAmount,
              currency: normalizedCurrency,
              category: "accommodation",
              source: "manual"
            }
          : null,
      notes: trimmedNotes || null
    });
  }

  function handlePlaceSelected(nextPlace: Place) {
    setPlace(nextPlace);
    setName(nextPlace.name);
    setAddress(nextPlace.address);
    setType(typeFromPlace(nextPlace));
  }

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_9rem]">
        <div className="grid gap-1">
          <label className="text-sm font-medium text-slate-700" htmlFor="accommodation-name">
            Name
          </label>
          <Input
            disabled={isSaving}
            id="accommodation-name"
            maxLength={200}
            onChange={(event) => setName(event.target.value)}
            placeholder="Hotel Roma"
            value={name}
          />
        </div>
        <div className="grid gap-1">
          <label className="text-sm font-medium text-slate-700" htmlFor="accommodation-type">
            Type
          </label>
          <Select
            disabled={isSaving}
            id="accommodation-type"
            onChange={(event) => setType(event.target.value as AccommodationType)}
            value={type}
          >
            {ACCOMMODATION_TYPES.map((option) => (
              <option key={option} value={option}>
                {formatType(option)}
              </option>
            ))}
          </Select>
        </div>
      </div>

      <div className="grid gap-1">
        <label className="text-sm font-medium text-slate-700" htmlFor="accommodation-address">
          Address
        </label>
        <Input
          disabled={isSaving}
          id="accommodation-address"
          maxLength={500}
          onChange={(event) => setAddress(event.target.value)}
          placeholder="Street address"
          value={address}
        />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button
          disabled={isSaving}
          onClick={() => setIsPlaceDialogOpen(true)}
          size="sm"
          type="button"
          variant="secondary"
        >
          {place ? "Change attached place" : "Attach place"}
        </Button>
        {place ? (
          <>
            <span className="text-sm text-slate-600">{place.name}</span>
            <Button
              disabled={isSaving}
              onClick={() => setPlace(null)}
              size="sm"
              type="button"
              variant="ghost"
            >
              Remove place
            </Button>
          </>
        ) : null}
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <div className="grid gap-1">
          <label className="text-sm font-medium text-slate-700" htmlFor="accommodation-check-in">
            Check-in
          </label>
          <Input
            disabled={isSaving}
            id="accommodation-check-in"
            onChange={(event) => setCheckInDate(event.target.value)}
            type="date"
            value={checkInDate}
          />
        </div>
        <div className="grid gap-1">
          <label className="text-sm font-medium text-slate-700" htmlFor="accommodation-check-out">
            Check-out
          </label>
          <Input
            disabled={isSaving}
            id="accommodation-check-out"
            onChange={(event) => setCheckOutDate(event.target.value)}
            type="date"
            value={checkOutDate}
          />
        </div>
      </div>

      <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_6rem]">
        <div className="grid gap-1">
          <label className="text-sm font-medium text-slate-700" htmlFor="accommodation-cost">
            Estimated stay cost
          </label>
          <Input
            disabled={isSaving}
            id="accommodation-cost"
            inputMode="decimal"
            min={0}
            onChange={(event) => setAmount(event.target.value)}
            placeholder="420"
            step="0.01"
            type="number"
            value={amount}
          />
        </div>
        <div className="grid gap-1">
          <label className="text-sm font-medium text-slate-700" htmlFor="accommodation-currency">
            Currency
          </label>
          <Input
            disabled={isSaving}
            id="accommodation-currency"
            maxLength={3}
            onChange={(event) => setCurrency(event.target.value.toUpperCase())}
            placeholder="EUR"
            value={currency}
          />
        </div>
      </div>

      <div className="grid gap-1">
        <label className="text-sm font-medium text-slate-700" htmlFor="accommodation-notes">
          Notes
        </label>
        <Textarea
          disabled={isSaving}
          id="accommodation-notes"
          maxLength={1000}
          onChange={(event) => setNotes(event.target.value)}
          placeholder="Late arrival, luggage storage, room preferences"
          value={notes}
        />
      </div>

      {error ? <p className="text-sm text-red-700">{error}</p> : null}

      <div className="flex flex-wrap gap-2">
        <Button disabled={isSaving} onClick={handleSave} size="sm" type="button">
          {isSaving ? "Saving..." : "Save accommodation"}
        </Button>
        <Button disabled={isSaving} onClick={onCancel} size="sm" type="button" variant="ghost">
          Cancel
        </Button>
        {initial && onClear ? (
          <Button disabled={isSaving} onClick={onClear} size="sm" type="button" variant="ghost">
            Clear accommodation
          </Button>
        ) : null}
      </div>

      <AttachPlaceDialog
        destination={destination}
        initialQuery={name || destination || ""}
        onClose={() => setIsPlaceDialogOpen(false)}
        onSelect={handlePlaceSelected}
        open={isPlaceDialogOpen}
      />
    </div>
  );
}

function typeFromPlace(place: Place): AccommodationType {
  const category = (place.category ?? "").toLowerCase();
  const name = place.name.toLowerCase();
  if (category.includes("hostel") || name.includes("hostel")) {
    return "hostel";
  }
  if (category.includes("apartment") || name.includes("apartment")) {
    return "apartment";
  }
  if (category.includes("guesthouse") || name.includes("guest house")) {
    return "guesthouse";
  }
  if (category.includes("home") || name.includes("home")) {
    return "home";
  }
  if (category.includes("hotel") || category.includes("lodging") || name.includes("hotel")) {
    return "hotel";
  }
  return "other";
}

function formatType(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
