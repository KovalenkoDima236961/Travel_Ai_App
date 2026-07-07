import { useState } from "react";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Textarea } from "@/shared/ui/textarea";
import type {
  TripTemplateDetail,
  UpdateTripTemplateInput
} from "@/entities/trip-template/model";

type TemplateMetadataDialogProps = {
  open: boolean;
  disabled: boolean;
  template: TripTemplateDetail;
  onClose: () => void;
  onSubmit: (input: UpdateTripTemplateInput) => void;
};

export function TemplateMetadataDialog({
  open,
  disabled,
  template,
  onClose,
  onSubmit
}: TemplateMetadataDialogProps) {
  const [title, setTitle] = useState(template.title);
  const [description, setDescription] = useState(template.description ?? "");
  const [destinationHint, setDestinationHint] = useState(template.destinationHint ?? "");
  const [defaultCurrency, setDefaultCurrency] = useState(template.defaultCurrency ?? "");
  const [tags, setTags] = useState(template.tags.join(", "));

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4">
      <div className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-lg bg-white p-6 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <h2 className="text-xl font-semibold text-slate-950">Edit template metadata</h2>
          <Button disabled={disabled} onClick={onClose} type="button" variant="ghost">
            Close
          </Button>
        </div>
        <form
          className="mt-6 space-y-5"
          onSubmit={(event) => {
            event.preventDefault();
            onSubmit({
              title,
              description: description.trim() || null,
              destinationHint: destinationHint.trim() || null,
              defaultCurrency: defaultCurrency.trim().toUpperCase() || null,
              tags: tags
                .split(",")
                .map((tag) => tag.trim())
                .filter(Boolean)
            });
          }}
        >
          <label className="block text-sm font-medium text-slate-700">
            Title
            <Input className="mt-2" onChange={(event) => setTitle(event.target.value)} value={title} />
          </label>
          <label className="block text-sm font-medium text-slate-700">
            Description
            <Textarea
              className="mt-2"
              onChange={(event) => setDescription(event.target.value)}
              value={description}
            />
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
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
              <Input
                className="mt-2"
                maxLength={3}
                onChange={(event) => setDefaultCurrency(event.target.value)}
                value={defaultCurrency}
              />
            </label>
          </div>
          <label className="block text-sm font-medium text-slate-700">
            Tags
            <Input className="mt-2" onChange={(event) => setTags(event.target.value)} value={tags} />
          </label>
          <div className="flex flex-wrap justify-end gap-2">
            <Button disabled={disabled} onClick={onClose} type="button" variant="secondary">
              Cancel
            </Button>
            <Button disabled={disabled} type="submit">
              {disabled ? "Saving..." : "Save changes"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
