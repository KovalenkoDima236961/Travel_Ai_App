import { apiFetch } from "@/shared/api/client";
import type { Trip } from "@/entities/trip/model";
import type {
  CreateTripFromTemplateInput,
  DuplicateTripTemplateInput,
  ListTripTemplatesParams,
  ListTripTemplatesResponse,
  SaveTripAsTemplateInput,
  TripTemplateDetail,
  UpdateTripTemplateInput
} from "@/entities/trip-template/model";

export const tripTemplateKeys = {
  all: ["trip-templates"] as const,
  lists: () => [...tripTemplateKeys.all, "list"] as const,
  list: (params: ListTripTemplatesParams) => [...tripTemplateKeys.lists(), params] as const,
  workspace: (workspaceId: string, params: ListTripTemplatesParams = {}) =>
    [...tripTemplateKeys.all, "workspace", workspaceId, params] as const,
  details: () => [...tripTemplateKeys.all, "detail"] as const,
  detail: (templateId: string) => [...tripTemplateKeys.details(), templateId] as const
};

export function listTripTemplates(params: ListTripTemplatesParams = {}) {
  const query = buildTemplateQuery(params);
  return apiFetch<ListTripTemplatesResponse>(`/trip-templates${query}`);
}

export function listWorkspaceTripTemplates(
  workspaceId: string,
  params: Omit<ListTripTemplatesParams, "workspaceId" | "visibility"> = {}
) {
  const query = buildTemplateQuery(params);
  return apiFetch<ListTripTemplatesResponse>(`/workspaces/${workspaceId}/templates${query}`);
}

export function getTripTemplate(templateId: string) {
  return apiFetch<TripTemplateDetail>(`/trip-templates/${templateId}`);
}

export function saveTripAsTemplate(tripId: string, input: SaveTripAsTemplateInput) {
  return apiFetch<TripTemplateDetail>(`/trips/${tripId}/templates`, {
    method: "POST",
    body: JSON.stringify(cleanSaveTemplatePayload(input))
  });
}

export function updateTripTemplate(templateId: string, input: UpdateTripTemplateInput) {
  return apiFetch<TripTemplateDetail>(`/trip-templates/${templateId}`, {
    method: "PATCH",
    body: JSON.stringify(cleanUpdateTemplatePayload(input))
  });
}

export function archiveTripTemplate(templateId: string, reason?: string) {
  return apiFetch<TripTemplateDetail>(`/trip-templates/${templateId}/archive`, {
    method: "POST",
    body: JSON.stringify(reason ? { reason } : {})
  });
}

export function duplicateTripTemplate(templateId: string, input: DuplicateTripTemplateInput) {
  return apiFetch<TripTemplateDetail>(`/trip-templates/${templateId}/duplicate`, {
    method: "POST",
    body: JSON.stringify(cleanDuplicateTemplatePayload(input))
  });
}

export function createTripFromTemplate(
  templateId: string,
  input: CreateTripFromTemplateInput
) {
  return apiFetch<Trip>(`/trip-templates/${templateId}/create-trip`, {
    method: "POST",
    body: JSON.stringify(cleanCreateTripFromTemplatePayload(input))
  });
}

function buildTemplateQuery(params: ListTripTemplatesParams) {
  const searchParams = new URLSearchParams();
  if (params.visibility) {
    searchParams.set("visibility", params.visibility);
  }
  if (params.workspaceId) {
    searchParams.set("workspaceId", params.workspaceId);
  }
  if (params.status) {
    searchParams.set("status", params.status);
  }
  if (params.tag) {
    searchParams.set("tag", params.tag);
  }
  if (params.q) {
    searchParams.set("q", params.q);
  }
  if (params.limit != null) {
    searchParams.set("limit", String(params.limit));
  }
  if (params.offset != null) {
    searchParams.set("offset", String(params.offset));
  }
  const query = searchParams.toString();
  return query ? `?${query}` : "";
}

function cleanSaveTemplatePayload(input: SaveTripAsTemplateInput) {
  return {
    title: input.title.trim(),
    description: input.description?.trim() || null,
    visibility: input.visibility,
    ...(input.visibility === "workspace" && input.workspaceId
      ? { workspaceId: input.workspaceId }
      : {}),
    destinationHint: input.destinationHint?.trim() || null,
    defaultCurrency: input.defaultCurrency?.trim().toUpperCase() || null,
    tags: normalizeTags(input.tags ?? [])
  };
}

function cleanUpdateTemplatePayload(input: UpdateTripTemplateInput) {
  return {
    ...(input.title !== undefined ? { title: input.title.trim() } : {}),
    ...(input.description !== undefined ? { description: input.description?.trim() ?? "" } : {}),
    ...(input.destinationHint !== undefined
      ? { destinationHint: input.destinationHint?.trim() ?? "" }
      : {}),
    ...(input.defaultCurrency !== undefined
      ? { defaultCurrency: input.defaultCurrency?.trim().toUpperCase() ?? "" }
      : {}),
    ...(input.tags !== undefined ? { tags: normalizeTags(input.tags) } : {})
  };
}

function cleanDuplicateTemplatePayload(input: DuplicateTripTemplateInput) {
  return {
    title: input.title?.trim() || "",
    visibility: input.visibility,
    ...(input.visibility === "workspace" && input.workspaceId
      ? { workspaceId: input.workspaceId }
      : {})
  };
}

function cleanCreateTripFromTemplatePayload(input: CreateTripFromTemplateInput) {
  return {
    title: input.title.trim(),
    destination: input.destination.trim(),
    startDate: input.startDate,
    ...(input.workspaceId ? { workspaceId: input.workspaceId } : {}),
    ...(input.budget?.amount != null
      ? {
          budget: {
            amount: input.budget.amount,
            currency: input.budget.currency.trim().toUpperCase()
          }
        }
      : {}),
    ...(input.travelers != null ? { travelers: input.travelers } : {}),
    ...(input.pace ? { pace: input.pace } : {})
  };
}

function normalizeTags(tags: string[]) {
  return tags
    .flatMap((tag) => tag.split(","))
    .map((tag) => tag.trim())
    .filter(Boolean);
}
