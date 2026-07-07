"use client";

import { FormEvent, useEffect, useState } from "react";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { useTripTemplateMutations } from "../model/useTripTemplates";
import { getErrorMessage } from "@/lib/utils";
import type { Trip } from "@/entities/trip/model";
import type { TripTemplateDetail, TripTemplateVisibility } from "@/entities/trip-template/model";

type SaveTripAsTemplateDialogProps = {
  open: boolean;
  trip: Trip;
  onClose: () => void;
  onSaved?: (template: TripTemplateDetail) => void;
};

export function SaveTripAsTemplateDialog({
  open,
  trip,
  onClose,
  onSaved
}: SaveTripAsTemplateDialogProps) {
  const { editableWorkspaces } = useWorkspaces();
  const mutations = useTripTemplateMutations();
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [visibility, setVisibility] = useState<TripTemplateVisibility>("private");
  const [workspaceId, setWorkspaceId] = useState("");
  const [destinationHint, setDestinationHint] = useState("");
  const [defaultCurrency, setDefaultCurrency] = useState("EUR");
  const [tags, setTags] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    setTitle(`${trip.destination} template`);
    setDescription("");
    setVisibility(trip.workspaceId ? "workspace" : "private");
    setWorkspaceId(trip.workspaceId ?? editableWorkspaces[0]?.id ?? "");
    setDestinationHint(trip.destination);
    setDefaultCurrency(trip.budgetCurrency || "EUR");
    setTags(trip.interests.join(", "));
    setLocalError(null);
  }, [editableWorkspaces, open, trip]);

  if (!open) {
    return null;
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (title.trim().length < 2) {
      setLocalError("Title must be at least 2 characters.");
      return;
    }
    if (visibility === "workspace" && !workspaceId) {
      setLocalError("Choose a workspace.");
      return;
    }
    try {
      setLocalError(null);
      const template = await mutations.saveTripAsTemplate.mutateAsync({
        tripId: trip.id,
        input: {
          title,
          description,
          visibility,
          workspaceId: visibility === "workspace" ? workspaceId : null,
          destinationHint,
          defaultCurrency,
          tags: splitTags(tags)
        }
      });
      onSaved?.(template);
      onClose();
    } catch (error) {
      setLocalError(getErrorMessage(error, "Could not save template."));
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4">
      <Card className="max-h-[90vh] w-full max-w-2xl overflow-y-auto">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-xl font-semibold text-slate-950">Save as template</h2>
            <p className="mt-1 text-sm text-slate-600">
              Live availability, comments, collaborators, share links, and calendar metadata are not copied.
            </p>
          </div>
          <Button onClick={onClose} type="button" variant="ghost">
            Close
          </Button>
        </div>
        <form className="mt-6 space-y-5" onSubmit={submit}>
          <label className="block text-sm font-medium text-slate-700">
            Title
            <Input className="mt-2" onChange={(event) => setTitle(event.target.value)} value={title} />
          </label>
          <label className="block text-sm font-medium text-slate-700">
            Description
            <Textarea
              className="mt-2"
              maxLength={1000}
              onChange={(event) => setDescription(event.target.value)}
              value={description}
            />
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm font-medium text-slate-700">
              Visibility
              <Select
                className="mt-2"
                onChange={(event) => setVisibility(event.target.value as TripTemplateVisibility)}
                value={visibility}
              >
                <option value="private">Private</option>
                {editableWorkspaces.length > 0 ? <option value="workspace">Workspace</option> : null}
              </Select>
            </label>
            {visibility === "workspace" ? (
              <label className="block text-sm font-medium text-slate-700">
                Workspace
                <Select
                  className="mt-2"
                  onChange={(event) => setWorkspaceId(event.target.value)}
                  value={workspaceId}
                >
                  <option value="">Choose workspace</option>
                  {editableWorkspaces.map((workspace) => (
                    <option key={workspace.id} value={workspace.id}>
                      {workspace.name}
                    </option>
                  ))}
                </Select>
              </label>
            ) : null}
            <label className="block text-sm font-medium text-slate-700">
              Destination hint
              <Input
                className="mt-2"
                onChange={(event) => setDestinationHint(event.target.value)}
                value={destinationHint}
              />
            </label>
            <label className="block text-sm font-medium text-slate-700">
              Default currency
              <Select
                className="mt-2"
                onChange={(event) => setDefaultCurrency(event.target.value)}
                value={defaultCurrency}
              >
                {["EUR", "USD", "GBP", "CZK"].map((code) => (
                  <option key={code} value={code}>
                    {code}
                  </option>
                ))}
              </Select>
            </label>
          </div>
          <label className="block text-sm font-medium text-slate-700">
            Tags
            <Input className="mt-2" onChange={(event) => setTags(event.target.value)} value={tags} />
          </label>
          {localError ? (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
              {localError}
            </div>
          ) : null}
          <div className="flex flex-wrap justify-end gap-2">
            <Button onClick={onClose} type="button" variant="secondary">
              Cancel
            </Button>
            <Button disabled={mutations.saveTripAsTemplate.isPending} type="submit">
              {mutations.saveTripAsTemplate.isPending ? "Saving..." : "Save template"}
            </Button>
          </div>
        </form>
      </Card>
    </div>
  );
}

function splitTags(value: string) {
  return value
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);
}
