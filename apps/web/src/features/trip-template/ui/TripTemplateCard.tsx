"use client";

import Link from "next/link";
import { Button, buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { formatBudget, formatDate } from "@/lib/utils";
import type { TripTemplate } from "@/entities/trip-template/model";

type TripTemplateCardProps = {
  template: TripTemplate;
  onArchive?: (template: TripTemplate) => void;
  onDuplicate?: (template: TripTemplate) => void;
  onUse?: (template: TripTemplate) => void;
  onAdapt?: (template: TripTemplate) => void;
};

export function TripTemplateCard({
  template,
  onArchive,
  onDuplicate,
  onUse,
  onAdapt
}: TripTemplateCardProps) {
  return (
    <Card className="flex h-full flex-col gap-5 transition hover:-translate-y-0.5 hover:border-primary-100 hover:shadow-lg">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="break-words text-lg font-semibold text-slate-950">
            {template.title}
          </h3>
          <Badge>{template.visibility === "workspace" ? "Workspace" : "Private"}</Badge>
          {template.status === "archived" ? <Badge tone="slate">Archived</Badge> : null}
        </div>
        <p className="mt-2 text-sm leading-6 text-slate-600">
          {template.description ||
            template.destinationHint ||
            "Reusable itinerary structure."}
        </p>
      </div>

      <div className="grid grid-cols-2 gap-3 text-sm">
        <Fact label="Destination" value={template.destinationHint || "Flexible"} />
        <Fact
          label="Duration"
          value={`${template.durationDays} ${template.durationDays === 1 ? "day" : "days"}`}
        />
        <Fact
          label="Estimate"
          value={formatBudget(
            template.estimatedTotalAmount,
            template.estimatedTotalCurrency || template.defaultCurrency || "EUR"
          )}
        />
        <Fact label="Created" value={formatDate(template.createdAt)} />
      </div>

      {template.tags.length > 0 ? (
        <div className="flex flex-wrap gap-2">
          {template.tags.slice(0, 6).map((tag) => (
            <span
              className="rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-medium text-slate-700"
              key={tag}
            >
              {tag}
            </span>
          ))}
        </div>
      ) : null}

      <div className="mt-auto flex flex-wrap gap-2">
        <Link
          className={buttonStyles({ variant: "secondary", size: "sm" })}
          href={`/templates/${template.id}`}
        >
          View
        </Link>
        {template.access.canUse && onAdapt ? (
          <Button onClick={() => onAdapt(template)} size="sm" type="button">
            Adapt with AI
          </Button>
        ) : null}
        {template.access.canUse && onUse ? (
          <Button onClick={() => onUse(template)} size="sm" type="button" variant="secondary">
            Use template
          </Button>
        ) : null}
        {template.access.canDuplicate && onDuplicate ? (
          <Button
            onClick={() => onDuplicate(template)}
            size="sm"
            type="button"
            variant="secondary"
          >
            Duplicate
          </Button>
        ) : null}
        {template.access.canArchive && onArchive ? (
          <Button
            onClick={() => onArchive(template)}
            size="sm"
            type="button"
            variant="ghost"
          >
            Archive
          </Button>
        ) : null}
      </div>
    </Card>
  );
}

function Badge({
  children,
  tone = "primary"
}: {
  children: string;
  tone?: "primary" | "slate";
}) {
  return (
    <span
      className={
        tone === "primary"
          ? "rounded-full bg-primary-50 px-2.5 py-1 text-xs font-semibold text-primary-700"
          : "rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-700"
      }
    >
      {children}
    </span>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs font-medium text-slate-500">{label}</p>
      <p className="mt-1 break-words font-semibold text-slate-800">{value}</p>
    </div>
  );
}
