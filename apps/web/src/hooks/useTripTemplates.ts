"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  archiveTripTemplate,
  createTripFromTemplate,
  duplicateTripTemplate,
  listTripTemplates,
  saveTripAsTemplate,
  tripTemplateKeys,
  updateTripTemplate
} from "@/lib/api/trip-templates";
import { tripKeys } from "@/lib/api/trips";
import type {
  CreateTripFromTemplateInput,
  DuplicateTripTemplateInput,
  ListTripTemplatesParams,
  SaveTripAsTemplateInput,
  UpdateTripTemplateInput
} from "@/types/trip-template";

export function useTripTemplates(params: ListTripTemplatesParams = {}) {
  return useQuery({
    queryKey: tripTemplateKeys.list(params),
    queryFn: () => listTripTemplates(params)
  });
}

export function useTripTemplateMutations() {
  const queryClient = useQueryClient();

  async function invalidateTemplates() {
    await queryClient.invalidateQueries({ queryKey: tripTemplateKeys.all });
  }

  return {
    saveTripAsTemplate: useMutation({
      mutationFn: ({ tripId, input }: { tripId: string; input: SaveTripAsTemplateInput }) =>
        saveTripAsTemplate(tripId, input),
      onSuccess: invalidateTemplates
    }),
    updateTemplate: useMutation({
      mutationFn: ({
        templateId,
        input
      }: {
        templateId: string;
        input: UpdateTripTemplateInput;
      }) => updateTripTemplate(templateId, input),
      onSuccess: async (template) => {
        queryClient.setQueryData(tripTemplateKeys.detail(template.id), template);
        await invalidateTemplates();
      }
    }),
    archiveTemplate: useMutation({
      mutationFn: ({ templateId, reason }: { templateId: string; reason?: string }) =>
        archiveTripTemplate(templateId, reason),
      onSuccess: invalidateTemplates
    }),
    duplicateTemplate: useMutation({
      mutationFn: ({
        templateId,
        input
      }: {
        templateId: string;
        input: DuplicateTripTemplateInput;
      }) => duplicateTripTemplate(templateId, input),
      onSuccess: invalidateTemplates
    }),
    createTripFromTemplate: useMutation({
      mutationFn: ({
        templateId,
        input
      }: {
        templateId: string;
        input: CreateTripFromTemplateInput;
      }) => createTripFromTemplate(templateId, input),
      onSuccess: async (trip) => {
        await queryClient.invalidateQueries({ queryKey: tripKeys.lists() });
        queryClient.setQueryData(tripKeys.detail(trip.id), trip);
      }
    })
  };
}
