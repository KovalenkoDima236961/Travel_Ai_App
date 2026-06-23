"use client";

import { FormEvent, useEffect, useState } from "react";
import { searchPlaces } from "@/lib/api/places";
import type { Place } from "@/types/place";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";

type AttachPlaceDialogProps = {
  open: boolean;
  onClose: () => void;
  destination?: string;
  initialQuery?: string;
  onSelect: (place: Place) => void;
};

export function AttachPlaceDialog({
  open,
  onClose,
  destination,
  initialQuery = "",
  onSelect
}: AttachPlaceDialogProps) {
  const [query, setQuery] = useState(initialQuery);
  const [items, setItems] = useState<Place[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searched, setSearched] = useState(false);

  useEffect(() => {
    if (!open) {
      return;
    }

    const nextQuery = initialQuery.trim();
    setQuery(nextQuery);
    setItems([]);
    setError(null);
    setSearched(false);

    if (!nextQuery) {
      return;
    }

    let cancelled = false;
    setIsLoading(true);
    searchPlaces({ query: nextQuery, destination })
      .then((response) => {
        if (!cancelled) {
          setItems(response.items);
          setSearched(true);
        }
      })
      .catch((searchError) => {
        if (!cancelled) {
          setError(searchError instanceof Error ? searchError.message : "Could not search places.");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [destination, initialQuery, open]);

  if (!open) {
    return null;
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const trimmedQuery = query.trim();
    if (!trimmedQuery) {
      setError("Enter a place name to search.");
      setItems([]);
      setSearched(false);
      return;
    }

    try {
      setIsLoading(true);
      setError(null);
      const response = await searchPlaces({ query: trimmedQuery, destination });
      setItems(response.items);
      setSearched(true);
    } catch (searchError) {
      setError(searchError instanceof Error ? searchError.message : "Could not search places.");
      setItems([]);
      setSearched(true);
    } finally {
      setIsLoading(false);
    }
  }

  function selectPlace(place: Place) {
    onSelect(place);
    onClose();
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-950/40 px-4 py-10">
      <div
        aria-modal="true"
        className="w-full max-w-2xl rounded-lg bg-white p-5 shadow-xl"
        role="dialog"
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-950">Attach real place</h2>
            {destination ? (
              <p className="mt-1 text-sm text-slate-600">Searching in {destination}</p>
            ) : null}
          </div>
          <Button onClick={onClose} type="button" variant="ghost">
            Close
          </Button>
        </div>

        <form className="mt-5 grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto]" onSubmit={handleSubmit}>
          <Input
            aria-label="Place search query"
            disabled={isLoading}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search for a place"
            value={query}
          />
          <Button disabled={isLoading} type="submit">
            {isLoading ? "Searching..." : "Search"}
          </Button>
        </form>

        {error ? (
          <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
            {error}
          </div>
        ) : null}

        <div className="mt-5 space-y-3">
          {items.map((place) => (
            <div
              className="rounded-lg border border-slate-200 p-4"
              key={`${place.provider}-${place.providerPlaceId}`}
            >
              <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0">
                  <h3 className="font-semibold text-slate-950">{place.name}</h3>
                  <p className="mt-1 text-sm leading-5 text-slate-600">{place.address}</p>
                  <p className="mt-2 text-xs font-medium uppercase text-slate-500">
                    {place.category ? formatCategory(place.category) : "Place"}
                    {place.rating != null ? ` · Rating ${place.rating}` : ""}
                    {place.ratingCount != null ? ` (${place.ratingCount.toLocaleString()})` : ""}
                  </p>
                </div>
                <Button onClick={() => selectPlace(place)} size="sm" type="button">
                  Select
                </Button>
              </div>
            </div>
          ))}

          {!isLoading && searched && items.length === 0 && !error ? (
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
              No matching places found.
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function formatCategory(value: string) {
  return value
    .split(/[_\s-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}
