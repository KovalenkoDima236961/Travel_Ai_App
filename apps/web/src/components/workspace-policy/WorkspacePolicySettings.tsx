"use client";

import { useEffect, useState, type ReactNode } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";

import { canManageWorkspace } from "@/components/workspaces/WorkspaceProvider";
import { useWorkspacePolicy } from "@/hooks/useWorkspacePolicy";
import { getWorkspace, workspaceKeys } from "@/lib/api/workspaces";
import { getErrorMessage } from "@/lib/utils";
import { Button, buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import type {
  PolicySeverity,
  WorkspacePolicyRuleDocument,
  WorkspacePolicyRules
} from "@/types/workspace-policy";

const FALLBACK_DEFAULTS: WorkspacePolicyRuleDocument = {
  schemaVersion: 1,
  rules: {
    requireTripBudget: { enabled: false, severity: "warning" },
    maxTripBudget: { enabled: false, severity: "blocking", amount: 1500, currency: "EUR" },
    maxDailyBudget: { enabled: false, severity: "warning", amount: 250, currency: "EUR" },
    maxItemCost: {
      enabled: false,
      severity: "warning",
      amount: 100,
      currency: "EUR",
      categories: []
    },
    maxAccommodationTotal: {
      enabled: false,
      severity: "warning",
      amount: 800,
      currency: "EUR"
    },
    maxAccommodationPerNight: {
      enabled: false,
      severity: "warning",
      amount: 120,
      currency: "EUR"
    },
    requireCostSplitting: { enabled: false, severity: "warning" },
    requireAvailabilityForTicketedItems: { enabled: true, severity: "warning" },
    maxWalkingKmPerDay: { enabled: true, severity: "warning", km: 12 },
    noLateActivitiesAfter: { enabled: true, severity: "warning", time: "22:00" },
    requiredRestTimePerDay: { enabled: false, severity: "info", minutes: 60 },
    preferredTransportModes: { enabled: false, severity: "info", modes: [] },
    disallowedActivityTypes: { enabled: false, severity: "warning", types: [] }
  }
};

export function WorkspacePolicySettings() {
  const params = useParams<{ workspaceId: string }>();
  const workspaceId = params.workspaceId;
  const workspace = useQuery({
    queryKey: workspaceKeys.detail(workspaceId),
    queryFn: () => getWorkspace(workspaceId),
    enabled: Boolean(workspaceId)
  });
  const { query, upsert, archive } = useWorkspacePolicy(workspaceId);
  const [name, setName] = useState("Default planning policy");
  const [description, setDescription] = useState("");
  const [document, setDocument] = useState<WorkspacePolicyRuleDocument>(FALLBACK_DEFAULTS);
  const [validationError, setValidationError] = useState<string | null>(null);
  const canEdit = workspace.data ? canManageWorkspace(workspace.data.currentUserRole) : false;

  useEffect(() => {
    if (!query.data) {
      return;
    }
    if (query.data.policy) {
      setName(query.data.policy.name);
      setDescription(query.data.policy.description ?? "");
      setDocument(query.data.policy.rules);
    } else {
      setDocument(query.data.defaults ?? FALLBACK_DEFAULTS);
    }
  }, [query.data]);

  function patchRule<K extends keyof WorkspacePolicyRules>(
    key: K,
    patch: Partial<WorkspacePolicyRules[K]>
  ) {
    setDocument((current) => ({
      ...current,
      rules: {
        ...current.rules,
        [key]: { ...current.rules[key], ...patch }
      }
    }));
  }

  function validate(): string | null {
    if (name.trim().length < 2 || name.trim().length > 100) {
      return "Policy name must be between 2 and 100 characters.";
    }
    if (description.trim().length > 500) {
      return "Description must be at most 500 characters.";
    }
    const moneyRules = [
      document.rules.maxTripBudget,
      document.rules.maxDailyBudget,
      document.rules.maxItemCost,
      document.rules.maxAccommodationTotal,
      document.rules.maxAccommodationPerNight
    ];
    if (moneyRules.some((rule) => rule.amount < 0)) {
      return "Amounts must be greater than or equal to zero.";
    }
    if (moneyRules.some((rule) => rule.enabled && !/^[A-Z]{3}$/.test(rule.currency))) {
      return "Enabled money rules require a 3-letter uppercase currency.";
    }
    if (
      document.rules.noLateActivitiesAfter.enabled &&
      !/^(?:[01]\d|2[0-3]):[0-5]\d$/.test(document.rules.noLateActivitiesAfter.time)
    ) {
      return "Late activity time must use HH:mm.";
    }
    if (document.rules.maxWalkingKmPerDay.enabled && document.rules.maxWalkingKmPerDay.km <= 0) {
      return "Walking distance must be greater than zero.";
    }
    if (document.rules.requiredRestTimePerDay.minutes < 0) {
      return "Rest time must be greater than or equal to zero.";
    }
    if (
      [
        document.rules.maxItemCost.categories,
        document.rules.preferredTransportModes.modes,
        document.rules.disallowedActivityTypes.types
      ].some((values) => values.length > 30)
    ) {
      return "Rule lists may contain at most 30 values.";
    }
    return null;
  }

  function save() {
    const error = validate();
    setValidationError(error);
    if (error) {
      return;
    }
    upsert.mutate({
      name: name.trim(),
      description: description.trim() || null,
      rules: document
    });
  }

  function archivePolicy() {
    if (window.confirm("Archive this policy? Workspace trips will have no active policy.")) {
      archive.mutate();
    }
  }

  if (query.isLoading || workspace.isLoading) {
    return <p className="text-sm text-slate-600">Loading workspace policy…</p>;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-sm font-semibold uppercase text-primary-700">Workspace settings</p>
          <h1 className="mt-2 text-3xl font-semibold text-slate-950">Planning policy</h1>
          <p className="mt-2 max-w-2xl text-sm text-slate-600">
            Policy rules guide planning and approval review. They are not legal or compliance rules.
          </p>
        </div>
        <Link className={buttonStyles({ variant: "secondary" })} href={`/workspaces/${workspaceId}/settings`}>
          Back to settings
        </Link>
      </div>

      {query.isError || workspace.isError ? (
        <Card><p className="text-sm text-red-700">Could not load workspace policy.</p></Card>
      ) : null}

      <Card className="space-y-5">
        <label className="block">
          <span className="text-sm font-medium text-slate-800">Policy name</span>
          <Input
            className="mt-2"
            disabled={!canEdit}
            maxLength={100}
            value={name}
            onChange={(event) => setName(event.target.value)}
          />
        </label>
        <label className="block">
          <span className="text-sm font-medium text-slate-800">Description</span>
          <Textarea
            className="mt-2"
            disabled={!canEdit}
            maxLength={500}
            value={description}
            onChange={(event) => setDescription(event.target.value)}
          />
        </label>
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        <RuleCard title="Require trip budget" rule={document.rules.requireTripBudget} disabled={!canEdit}
          onChange={(patch) => patchRule("requireTripBudget", patch)} />
        <MoneyRuleCard title="Maximum trip budget" rule={document.rules.maxTripBudget} disabled={!canEdit}
          onChange={(patch) => patchRule("maxTripBudget", patch)} />
        <MoneyRuleCard title="Maximum daily budget" rule={document.rules.maxDailyBudget} disabled={!canEdit}
          onChange={(patch) => patchRule("maxDailyBudget", patch)} />
        <MoneyRuleCard title="Maximum item cost" rule={document.rules.maxItemCost} disabled={!canEdit}
          onChange={(patch) => patchRule("maxItemCost", patch)}>
          <ListInput label="Categories" value={document.rules.maxItemCost.categories} disabled={!canEdit}
            onChange={(categories) => patchRule("maxItemCost", { categories })} />
        </MoneyRuleCard>
        <MoneyRuleCard title="Maximum accommodation total" rule={document.rules.maxAccommodationTotal}
          disabled={!canEdit} onChange={(patch) => patchRule("maxAccommodationTotal", patch)} />
        <MoneyRuleCard title="Maximum accommodation per night" rule={document.rules.maxAccommodationPerNight}
          disabled={!canEdit} onChange={(patch) => patchRule("maxAccommodationPerNight", patch)} />
        <RuleCard title="Require cost splitting" rule={document.rules.requireCostSplitting} disabled={!canEdit}
          onChange={(patch) => patchRule("requireCostSplitting", patch)} />
        <RuleCard title="Require ticketed-item availability" rule={document.rules.requireAvailabilityForTicketedItems}
          disabled={!canEdit} onChange={(patch) => patchRule("requireAvailabilityForTicketedItems", patch)} />
        <RuleCard title="Maximum walking per day" rule={document.rules.maxWalkingKmPerDay} disabled={!canEdit}
          onChange={(patch) => patchRule("maxWalkingKmPerDay", patch)}>
          <NumberInput label="Kilometres" value={document.rules.maxWalkingKmPerDay.km} min={0.1}
            disabled={!canEdit} onChange={(km) => patchRule("maxWalkingKmPerDay", { km })} />
        </RuleCard>
        <RuleCard title="No late activities after" rule={document.rules.noLateActivitiesAfter} disabled={!canEdit}
          onChange={(patch) => patchRule("noLateActivitiesAfter", patch)}>
          <label className="block text-sm text-slate-700">Time
            <Input className="mt-1" type="time" disabled={!canEdit}
              value={document.rules.noLateActivitiesAfter.time}
              onChange={(event) => patchRule("noLateActivitiesAfter", { time: event.target.value })} />
          </label>
        </RuleCard>
        <RuleCard title="Required rest time per day" rule={document.rules.requiredRestTimePerDay}
          disabled={!canEdit} onChange={(patch) => patchRule("requiredRestTimePerDay", patch)}>
          <NumberInput label="Minutes" value={document.rules.requiredRestTimePerDay.minutes} min={0}
            disabled={!canEdit} onChange={(minutes) => patchRule("requiredRestTimePerDay", { minutes })} />
        </RuleCard>
        <RuleCard title="Preferred transport modes" rule={document.rules.preferredTransportModes}
          disabled={!canEdit} onChange={(patch) => patchRule("preferredTransportModes", patch)}>
          <ListInput label="Modes" value={document.rules.preferredTransportModes.modes} disabled={!canEdit}
            onChange={(modes) => patchRule("preferredTransportModes", { modes })} />
        </RuleCard>
        <RuleCard title="Disallowed activity types" rule={document.rules.disallowedActivityTypes}
          disabled={!canEdit} onChange={(patch) => patchRule("disallowedActivityTypes", patch)}>
          <ListInput label="Types" value={document.rules.disallowedActivityTypes.types} disabled={!canEdit}
            onChange={(types) => patchRule("disallowedActivityTypes", { types })} />
        </RuleCard>
      </div>

      {validationError || upsert.isError || archive.isError ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          {validationError ??
            getErrorMessage(upsert.error ?? archive.error, "Could not update workspace policy.")}
        </div>
      ) : null}
      {upsert.isSuccess ? (
        <p className="text-sm text-emerald-700">Workspace policy saved.</p>
      ) : null}

      {canEdit ? (
        <div className="flex flex-wrap justify-end gap-3">
          {query.data?.policy ? (
            <Button variant="ghost" disabled={archive.isPending} onClick={archivePolicy}>
              Archive policy
            </Button>
          ) : null}
          <Button disabled={upsert.isPending} onClick={save}>
            {upsert.isPending ? "Saving…" : "Save policy"}
          </Button>
        </div>
      ) : (
        <p className="text-sm text-slate-500">Only workspace owners and admins can edit this policy.</p>
      )}
    </div>
  );
}

function RuleCard({
  title,
  rule,
  disabled,
  onChange,
  children
}: {
  title: string;
  rule: { enabled: boolean; severity: PolicySeverity };
  disabled: boolean;
  onChange: (patch: { enabled?: boolean; severity?: PolicySeverity }) => void;
  children?: ReactNode;
}) {
  return (
    <Card className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <h2 className="font-semibold text-slate-950">{title}</h2>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={rule.enabled} disabled={disabled}
            onChange={(event) => onChange({ enabled: event.target.checked })} />
          Enabled
        </label>
      </div>
      <label className="block text-sm text-slate-700">Severity
        <Select className="mt-1" value={rule.severity} disabled={disabled}
          onChange={(event) => onChange({ severity: event.target.value as PolicySeverity })}>
          <option value="info">Info</option>
          <option value="warning">Warning</option>
          <option value="blocking">Blocking</option>
        </Select>
      </label>
      {children}
    </Card>
  );
}

function MoneyRuleCard({
  rule,
  onChange,
  children,
  ...props
}: Omit<Parameters<typeof RuleCard>[0], "rule" | "onChange"> & {
  rule: MoneyRuleValue;
  onChange: (patch: Partial<MoneyRuleValue>) => void;
}) {
  return (
    <RuleCard {...props} rule={rule} onChange={onChange}>
      <div className="grid grid-cols-[1fr_100px] gap-3">
        <NumberInput label="Amount" value={rule.amount} min={0} disabled={props.disabled}
          onChange={(amount) => onChange({ amount })} />
        <label className="block text-sm text-slate-700">Currency
          <Input className="mt-1 uppercase" maxLength={3} disabled={props.disabled}
            value={rule.currency}
            onChange={(event) => onChange({ currency: event.target.value.toUpperCase() })} />
        </label>
      </div>
      {children}
    </RuleCard>
  );
}

type MoneyRuleValue = {
  enabled: boolean;
  severity: PolicySeverity;
  amount: number;
  currency: string;
};

function NumberInput({
  label,
  value,
  min,
  disabled,
  onChange
}: {
  label: string;
  value: number;
  min: number;
  disabled: boolean;
  onChange: (value: number) => void;
}) {
  return (
    <label className="block text-sm text-slate-700">{label}
      <Input className="mt-1" type="number" min={min} step="any" disabled={disabled}
        value={Number.isFinite(value) ? value : 0}
        onChange={(event) => onChange(Number(event.target.value))} />
    </label>
  );
}

function ListInput({
  label,
  value,
  disabled,
  onChange
}: {
  label: string;
  value: string[];
  disabled: boolean;
  onChange: (value: string[]) => void;
}) {
  return (
    <label className="block text-sm text-slate-700">{label}
      <Input className="mt-1" disabled={disabled} value={value.join(", ")}
        placeholder="Comma-separated values"
        onChange={(event) =>
          onChange(event.target.value.split(",").map((item) => item.trim()).filter(Boolean))
        } />
    </label>
  );
}
