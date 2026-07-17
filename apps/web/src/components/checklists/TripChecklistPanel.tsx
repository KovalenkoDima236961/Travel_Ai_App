"use client";

import { FormEvent, useEffect, useMemo, useState, type ReactNode } from "react";
import { useTranslations } from "next-intl";
import { EmptyState, ErrorState, SectionLoadingState } from "@/components/ui";
import { useAppLanguage } from "@/components/i18n/I18nProvider";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { useChecklistMutations } from "@/hooks/useChecklistMutations";
import { useTripChecklist } from "@/hooks/useTripChecklist";
import { cn, formatDate, getErrorMessage } from "@/lib/utils";
import {
  applyOfflineChecklistChecked,
  applyOfflineChecklistCreate,
  buildChecklistSummary,
  isOfflinePending
} from "@/lib/offline/cache-writer";
import { getCachedChecklist, putCachedChecklist } from "@/lib/offline/trip-cache";
import { enqueueCompanionMutation } from "@/lib/offline/sync-queue";
import type {
  ChecklistCategory,
  ChecklistItemPayload,
  ChecklistItemType,
  ChecklistPriority,
  ChecklistViewResponse,
  TripChecklistItem,
  UpdateChecklistItemPayload
} from "@/entities/checklist/model";
import {
  CHECKLIST_CATEGORIES,
  CHECKLIST_ITEM_TYPES,
  CHECKLIST_PRIORITIES
} from "@/entities/checklist/model";

type TripChecklistPanelProps = {
  tripId: string;
  canEdit: boolean;
  canCheck: boolean;
  currentUserId?: string | null;
  enabled?: boolean;
  offline?: boolean;
  userId?: string | null;
};

type FilterStatus = "all" | "unchecked" | "checked" | "mine" | "high";

type ItemFormState = {
  title: string;
  description: string;
  category: ChecklistCategory;
  itemType: ChecklistItemType;
  priority: ChecklistPriority;
  quantity: string;
  assignedToUserId: string;
  dueDate: string;
  reason: string;
};

type ChecklistT = (key: string, values?: Record<string, string | number>) => string;

const EMPTY_FORM: ItemFormState = {
  title: "",
  description: "",
  category: "other",
  itemType: "packing",
  priority: "medium",
  quantity: "",
  assignedToUserId: "",
  dueDate: "",
  reason: ""
};

export function TripChecklistPanel({
  tripId,
  canEdit,
  canCheck,
  currentUserId,
  enabled = true,
  offline = false,
  userId
}: TripChecklistPanelProps) {
  const t = useTranslations("checklist");
  const emptyT = useTranslations("emptyStates.checklist");
  const errorsT = useTranslations("errors");
  const loadingT = useTranslations("loading");
  const { language } = useAppLanguage();
  const query = useTripChecklist(tripId, { enabled: enabled && !offline });
  const mutations = useChecklistMutations(tripId);
  const [offlineData, setOfflineData] = useState<ChecklistViewResponse | null>(null);
  const [categoryFilter, setCategoryFilter] = useState<ChecklistCategory | "all">("all");
  const [statusFilter, setStatusFilter] = useState<FilterStatus>("all");
  const [generationCategory, setGenerationCategory] = useState<ChecklistCategory | "all">("all");
  const [generationInstructions, setGenerationInstructions] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<ItemFormState>(EMPTY_FORM);
  const [editingItemId, setEditingItemId] = useState<string | null>(null);
  const [localError, setLocalError] = useState<string | null>(null);

  useEffect(() => {
    if (!offline || !userId) {
      return;
    }
    let cancelled = false;
    getCachedChecklist(tripId, userId)
      .then((record) => {
        if (!cancelled) {
          setOfflineData(record?.checklist ?? null);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setOfflineData(null);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [offline, tripId, userId]);

  useEffect(() => {
    if (!offline && userId && query.data) {
      void putCachedChecklist({ tripId, userId, checklist: query.data });
    }
  }, [offline, query.data, tripId, userId]);

  const data = offline ? offlineData : query.data;
  const checklist = data?.checklist ?? null;
  const items = useMemo(
    () => [...(checklist?.items ?? [])].sort(compareChecklistItems),
    [checklist?.items]
  );
  const filteredItems = useMemo(
    () =>
      items.filter((item) => {
        if (categoryFilter !== "all" && item.category !== categoryFilter) {
          return false;
        }
        if (statusFilter === "checked") {
          return item.checked;
        }
        if (statusFilter === "unchecked") {
          return !item.checked;
        }
        if (statusFilter === "mine") {
          return Boolean(currentUserId) && item.assignedToUserId === currentUserId;
        }
        if (statusFilter === "high") {
          return !item.checked && (item.priority === "high" || item.priority === "critical");
        }
        return true;
      }),
    [categoryFilter, currentUserId, items, statusFilter]
  );
  const summary = data?.summary ?? buildSummary(items, currentUserId);
  const progress =
    summary.totalItems > 0 ? Math.round((summary.checkedItems / summary.totalItems) * 100) : 0;
  const busy =
    mutations.generateMutation.isPending ||
    mutations.createItemMutation.isPending ||
    mutations.updateItemMutation.isPending ||
    mutations.deleteItemMutation.isPending ||
    mutations.setCheckedMutation.isPending ||
    mutations.reorderMutation.isPending;

  if (!enabled && !offline) {
    return (
      <Card id="checklist" className="scroll-mt-24">
        <h2 className="text-lg font-semibold text-slate-950">{t("title")}</h2>
        <p className="mt-2 text-sm text-slate-600">{t("onlineOnly")}</p>
      </Card>
    );
  }

  function generateChecklist(replaceAiItems: boolean) {
    if (offline) {
      setLocalError("This action requires internet.");
      return;
    }
    setLocalError(null);
    mutations.generateMutation.mutate(
      {
        mode: generationCategory === "all" ? (replaceAiItems ? "full" : "add_missing") : "category",
        categories: generationCategory === "all" ? [] : [generationCategory],
        instructions: generationInstructions,
        preserveCheckedItems: true,
        preserveManualItems: true,
        replaceAiItems,
        outputLanguage: language
      },
      {
        onError: (error) =>
          setLocalError(getErrorMessage(error, t("errors.generate")))
      }
    );
  }

  function submitItem(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const title = form.title.trim();
    if (!title) {
      setLocalError(t("errors.titleRequired"));
      return;
    }
    setLocalError(null);

    if (offline) {
      void submitOfflineItem(title);
      return;
    }

    if (editingItemId) {
      mutations.updateItemMutation.mutate(
        { itemId: editingItemId, input: updatePayloadFromForm(form) },
        {
          onSuccess: resetForm,
          onError: (error) => setLocalError(getErrorMessage(error, t("errors.save")))
        }
      );
      return;
    }

    mutations.createItemMutation.mutate(createPayloadFromForm(form), {
      onSuccess: resetForm,
      onError: (error) => setLocalError(getErrorMessage(error, t("errors.save")))
    });
  }

  function editItem(item: TripChecklistItem) {
    setEditingItemId(item.id);
    setShowForm(true);
    setForm(formFromItem(item));
    setLocalError(null);
  }

  function removeItem(item: TripChecklistItem) {
    if (!window.confirm(t("confirmDelete"))) {
      return;
    }
    if (offline) {
      void removeOfflineItem(item);
      return;
    }
    mutations.deleteItemMutation.mutate(item.id, {
      onError: (error) => setLocalError(getErrorMessage(error, t("errors.delete")))
    });
  }

  function toggleItem(item: TripChecklistItem) {
    if (offline) {
      void toggleOfflineItem(item);
      return;
    }
    mutations.setCheckedMutation.mutate(
      { itemId: item.id, checked: !item.checked },
      {
        onError: (error) => setLocalError(getErrorMessage(error, t("errors.check")))
      }
    );
  }

  function moveItem(itemId: string, direction: -1 | 1) {
    if (offline) {
      setLocalError("This action requires internet.");
      return;
    }
    const ordered = [...items];
    const index = ordered.findIndex((item) => item.id === itemId);
    const nextIndex = index + direction;
    if (index < 0 || nextIndex < 0 || nextIndex >= ordered.length) {
      return;
    }
    [ordered[index], ordered[nextIndex]] = [ordered[nextIndex], ordered[index]];
    mutations.reorderMutation.mutate(ordered.map((item) => item.id), {
      onError: (error) => setLocalError(getErrorMessage(error, t("errors.reorder")))
    });
  }

  async function submitOfflineItem(title: string) {
    if (!userId) {
      setLocalError("Open this trip online once before changing checklist items offline.");
      return;
    }
    setLocalError(null);
    if (editingItemId) {
      const current = items.find((item) => item.id === editingItemId);
      if (!current || !isOfflinePending(current.metadata)) {
        setLocalError("This action requires internet.");
        return;
      }
      const nextItems = items.map((item) =>
        item.id === editingItemId
          ? {
              ...item,
              ...updatePayloadFromForm(form),
              title,
              updatedAt: new Date().toISOString()
            }
          : item
      );
      const response: ChecklistViewResponse = {
        checklist: checklist
          ? { ...checklist, items: nextItems, updatedAt: new Date().toISOString() }
          : null,
        summary: buildChecklistSummary(nextItems, currentUserId),
        canGenerate: false
      };
      await putCachedChecklist({ tripId, userId, checklist: response });
      setOfflineData(response);
      resetForm();
      return;
    }

    const payload = createPayloadFromForm({ ...form, title });
    const clientMutationId = createClientMutationId();
    const { response, localEntityId } = await applyOfflineChecklistCreate({
      tripId,
      userId,
      payload,
      currentUserId,
      clientMutationId
    });
    await enqueueCompanionMutation({
      tripId,
      userId,
      type: "checklist_item_create",
      entity: "checklist",
      payload: { localEntityId, input: payload },
      localEntityId,
      clientMutationId
    });
    setOfflineData(response);
    resetForm();
  }

  async function toggleOfflineItem(item: TripChecklistItem) {
    if (!userId) {
      setLocalError("Open this trip online once before changing checklist items offline.");
      return;
    }
    const checked = !item.checked;
    const clientMutationId = createClientMutationId();
    const response = await applyOfflineChecklistChecked({
      tripId,
      userId,
      itemId: item.id,
      checked,
      currentUserId,
      clientMutationId
    });
    await enqueueCompanionMutation({
      tripId,
      userId,
      type: checked ? "checklist_item_check" : "checklist_item_uncheck",
      entity: "checklist",
      payload: checked
        ? { itemId: item.id, checkedAt: new Date().toISOString() }
        : { itemId: item.id, uncheckedAt: new Date().toISOString() },
      clientMutationId
    });
    setOfflineData(response);
  }

  async function removeOfflineItem(item: TripChecklistItem) {
    if (!userId || !checklist || !isOfflinePending(item.metadata)) {
      setLocalError("This action requires internet.");
      return;
    }
    const nextItems = checklist.items.filter((candidate) => candidate.id !== item.id);
    const response: ChecklistViewResponse = {
      checklist: { ...checklist, items: nextItems, updatedAt: new Date().toISOString() },
      summary: buildChecklistSummary(nextItems, currentUserId),
      canGenerate: false
    };
    await putCachedChecklist({ tripId, userId, checklist: response });
    await enqueueCompanionMutation({
      tripId,
      userId,
      type: "checklist_item_delete_local",
      entity: "checklist",
      payload: { localEntityId: item.id },
      localEntityId: item.id
    });
    setOfflineData(response);
  }

  function resetForm() {
    setForm(EMPTY_FORM);
    setEditingItemId(null);
    setShowForm(false);
  }

  return (
    <Card id="checklist" className="scroll-mt-24">
      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-slate-950">{t("title")}</h2>
            <p className="mt-1 text-sm leading-6 text-slate-600">
              {checklist?.summary || t("description")}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            {canEdit ? (
              <>
                <Button
                  disabled={busy || offline}
                  onClick={() => generateChecklist(false)}
                  size="sm"
                  title={offline ? "This action requires internet." : undefined}
                  type="button"
                  variant={checklist ? "secondary" : "primary"}
                >
                  {mutations.generateMutation.isPending ? t("generating") : t("generate")}
                </Button>
                {checklist ? (
                  <Button
                    disabled={busy || offline}
                    onClick={() => generateChecklist(true)}
                    size="sm"
                    title={offline ? "This action requires internet." : undefined}
                    type="button"
                    variant="secondary"
                  >
                    {t("regenerate")}
                  </Button>
                ) : null}
                <Button
                  disabled={busy}
                  onClick={() => {
                    setShowForm((open) => !open);
                    setEditingItemId(null);
                    setForm(EMPTY_FORM);
                  }}
                  size="sm"
                  type="button"
                  variant="secondary"
                >
                  {showForm && !editingItemId ? t("hideAdd") : t("addItem")}
                </Button>
              </>
            ) : null}
          </div>
        </div>

        {query.isLoading ? (
          <SectionLoadingState compact label={loadingT("checklist")} />
        ) : null}
        {query.isError ? (
          <ErrorState
            compact
            description={errorsT("checklistLoadDescription")}
            developmentDetails={query.error instanceof Error ? query.error.message : undefined}
            retryAction={{ onRetry: () => void query.refetch(), pending: query.isFetching }}
            title={errorsT("checklistLoadTitle")}
          />
        ) : null}
        {localError ? (
          <p className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
            {localError}
          </p>
        ) : null}

        <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm font-medium text-slate-900">
              {t("progress", {
                checked: summary.checkedItems,
                total: summary.totalItems,
                percent: progress
              })}
            </p>
            <p className="text-xs text-slate-500">
              {t("highPriority", { count: summary.highPriorityUnchecked })}
            </p>
          </div>
          <div className="mt-3 h-2 overflow-hidden rounded-full bg-slate-200">
            <div
              aria-label={t("progress", {
                checked: summary.checkedItems,
                total: summary.totalItems,
                percent: progress
              })}
              aria-valuemax={100}
              aria-valuemin={0}
              aria-valuenow={progress}
              className="h-full rounded-full bg-emerald-600 transition-all"
              role="progressbar"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>

        {canEdit ? (
          <div className="grid gap-3 rounded-md border border-slate-200 bg-white p-3 md:grid-cols-[1fr_1fr_2fr]">
            <label className="text-sm font-medium text-slate-700">
              {t("generationCategory")}
              <Select
                className="mt-1"
                value={generationCategory}
                onChange={(event) =>
                  setGenerationCategory(event.target.value as ChecklistCategory | "all")
                }
              >
                <option value="all">{t("allCategories")}</option>
                {CHECKLIST_CATEGORIES.map((category) => (
                  <option key={category} value={category}>
                    {t(`categories.${category}`)}
                  </option>
                ))}
              </Select>
            </label>
            <label className="text-sm font-medium text-slate-700 md:col-span-2">
              {t("instructions")}
              <Input
                className="mt-1"
                maxLength={1000}
                onChange={(event) => setGenerationInstructions(event.target.value)}
                placeholder={t("instructionsPlaceholder")}
                value={generationInstructions}
              />
            </label>
          </div>
        ) : null}

        {showForm ? (
          <ChecklistItemForm
            busy={busy}
            canAssignToMe={Boolean(currentUserId)}
            currentUserId={currentUserId}
            editing={Boolean(editingItemId)}
            form={form}
            onCancel={resetForm}
            onChange={setForm}
            onSubmit={submitItem}
            t={t}
          />
        ) : null}

        <div className="grid gap-3 sm:grid-cols-2">
          <label className="text-sm font-medium text-slate-700">
            {t("categoryFilter")}
            <Select
              className="mt-1"
              onChange={(event) => setCategoryFilter(event.target.value as ChecklistCategory | "all")}
              value={categoryFilter}
            >
              <option value="all">{t("allCategories")}</option>
              {CHECKLIST_CATEGORIES.map((category) => (
                <option key={category} value={category}>
                  {t(`categories.${category}`)}
                </option>
              ))}
            </Select>
          </label>
          <label className="text-sm font-medium text-slate-700">
            {t("statusFilter")}
            <Select
              className="mt-1"
              onChange={(event) => setStatusFilter(event.target.value as FilterStatus)}
              value={statusFilter}
            >
              <option value="all">{t("filters.all")}</option>
              <option value="unchecked">{t("filters.unchecked")}</option>
              <option value="checked">{t("filters.checked")}</option>
              <option value="mine">{t("filters.mine")}</option>
              <option value="high">{t("filters.high")}</option>
            </Select>
          </label>
        </div>

        {!query.isLoading && items.length === 0 ? (
          <EmptyState
            compact
            description={canEdit ? emptyT("description") : emptyT("viewerDescription")}
            primaryAction={
              canEdit
                ? {
                    disabled: busy || offline,
                    disabledReason: offline ? t("onlineOnly") : undefined,
                    label: emptyT("action"),
                    onClick: () => generateChecklist(false)
                  }
                : undefined
            }
            title={emptyT("title")}
          />
        ) : null}

        {filteredItems.length > 0 ? (
          <ul className="space-y-3">
            {filteredItems.map((item) => {
              const canToggle =
                canCheck &&
                (canEdit || !item.assignedToUserId || item.assignedToUserId === currentUserId);
              const orderedIndex = items.findIndex((candidate) => candidate.id === item.id);
              return (
                <li
                  className={cn(
                    "scroll-mt-24 rounded-lg border bg-white p-3",
                    item.checked ? "border-emerald-200" : "border-slate-200"
                  )}
                  id={`checklist-item-${item.id}`}
                  key={item.id}
                >
                  <div className="flex flex-col gap-3 sm:flex-row sm:items-start">
                    <input
                      checked={item.checked}
                      className="mt-1 h-5 w-5 rounded border-slate-300 text-emerald-600 focus:ring-emerald-500"
                      disabled={!canToggle || busy}
                      onChange={() => toggleItem(item)}
                      type="checkbox"
                    />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <p
                          className={cn(
                            "font-medium text-slate-950",
                            item.checked && "text-slate-500 line-through"
                          )}
                        >
                          {item.quantity ? `${item.quantity}x ` : ""}
                          {item.title}
                        </p>
                        <Badge tone={priorityTone(item.priority)}>
                          {t(`priorities.${item.priority}`)}
                        </Badge>
                        <Badge>{t(`categories.${item.category}`)}</Badge>
                        {item.assignedToDisplayName || item.assignedToUserId ? (
                          <Badge>{item.assignedToDisplayName || item.assignedToUserId}</Badge>
                        ) : null}
                        {isOfflinePending(item.metadata) ? (
                          <Badge tone="warning">Pending sync</Badge>
                        ) : null}
                      </div>
                      {item.description ? (
                        <p className="mt-2 text-sm leading-6 text-slate-600">{item.description}</p>
                      ) : null}
                      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
                        <span>{t(`itemTypes.${item.itemType}`)}</span>
                        <span>{t(`sources.${item.source}`)}</span>
                        {item.dueDate ? <span>{t("due", { date: formatDate(item.dueDate) })}</span> : null}
                        {item.relatedDayNumber ? (
                          <span>{t("relatedDay", { day: item.relatedDayNumber })}</span>
                        ) : null}
                      </div>
                      {item.reason ? (
                        <p className="mt-2 text-xs leading-5 text-slate-500">{item.reason}</p>
                      ) : null}
                    </div>
                    <div className="flex flex-wrap gap-2 sm:justify-end">
                      {canEdit ? (
                        <>
                          <Button
                            disabled={busy || offline || orderedIndex <= 0}
                            onClick={() => moveItem(item.id, -1)}
                            size="sm"
                            title={offline ? "This action requires internet." : undefined}
                            type="button"
                            variant="ghost"
                          >
                            {t("up")}
                          </Button>
                          <Button
                            disabled={busy || offline || orderedIndex >= items.length - 1}
                            onClick={() => moveItem(item.id, 1)}
                            size="sm"
                            title={offline ? "This action requires internet." : undefined}
                            type="button"
                            variant="ghost"
                          >
                            {t("down")}
                          </Button>
                          <Button
                            disabled={busy || (offline && !isOfflinePending(item.metadata))}
                            onClick={() => editItem(item)}
                            size="sm"
                            title={
                              offline && !isOfflinePending(item.metadata)
                                ? "This action requires internet."
                                : undefined
                            }
                            type="button"
                            variant="secondary"
                          >
                            {t("edit")}
                          </Button>
                          <Button
                            disabled={busy || (offline && !isOfflinePending(item.metadata))}
                            onClick={() => removeItem(item)}
                            size="sm"
                            title={
                              offline && !isOfflinePending(item.metadata)
                                ? "This action requires internet."
                                : undefined
                            }
                            type="button"
                            variant="danger"
                          >
                            {t("delete")}
                          </Button>
                        </>
                      ) : null}
                    </div>
                  </div>
                </li>
              );
            })}
          </ul>
        ) : items.length > 0 ? (
          <p className="rounded-md border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
            {t("noFilterResults")}
          </p>
        ) : null}
      </div>
    </Card>
  );
}

function ChecklistItemForm({
  busy,
  canAssignToMe,
  currentUserId,
  editing,
  form,
  onCancel,
  onChange,
  onSubmit,
  t
}: {
  busy: boolean;
  canAssignToMe: boolean;
  currentUserId?: string | null;
  editing: boolean;
  form: ItemFormState;
  onCancel: () => void;
  onChange: (next: ItemFormState) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  t: ChecklistT;
}) {
  return (
    <form className="rounded-md border border-slate-200 bg-slate-50 p-3" onSubmit={onSubmit}>
      <div className="grid gap-3 md:grid-cols-2">
        <label className="text-sm font-medium text-slate-700">
          {t("fields.title")}
          <Input
            className="mt-1"
            disabled={busy}
            maxLength={120}
            onChange={(event) => onChange({ ...form, title: event.target.value })}
            value={form.title}
          />
        </label>
        <label className="text-sm font-medium text-slate-700">
          {t("fields.category")}
          <Select
            className="mt-1"
            disabled={busy}
            onChange={(event) =>
              onChange({ ...form, category: event.target.value as ChecklistCategory })
            }
            value={form.category}
          >
            {CHECKLIST_CATEGORIES.map((category) => (
              <option key={category} value={category}>
                {t(`categories.${category}`)}
              </option>
            ))}
          </Select>
        </label>
        <label className="text-sm font-medium text-slate-700">
          {t("fields.itemType")}
          <Select
            className="mt-1"
            disabled={busy}
            onChange={(event) =>
              onChange({ ...form, itemType: event.target.value as ChecklistItemType })
            }
            value={form.itemType}
          >
            {CHECKLIST_ITEM_TYPES.map((itemType) => (
              <option key={itemType} value={itemType}>
                {t(`itemTypes.${itemType}`)}
              </option>
            ))}
          </Select>
        </label>
        <label className="text-sm font-medium text-slate-700">
          {t("fields.priority")}
          <Select
            className="mt-1"
            disabled={busy}
            onChange={(event) =>
              onChange({ ...form, priority: event.target.value as ChecklistPriority })
            }
            value={form.priority}
          >
            {CHECKLIST_PRIORITIES.map((priority) => (
              <option key={priority} value={priority}>
                {t(`priorities.${priority}`)}
              </option>
            ))}
          </Select>
        </label>
        <label className="text-sm font-medium text-slate-700">
          {t("fields.quantity")}
          <Input
            className="mt-1"
            disabled={busy}
            min={1}
            onChange={(event) => onChange({ ...form, quantity: event.target.value })}
            type="number"
            value={form.quantity}
          />
        </label>
        <label className="text-sm font-medium text-slate-700">
          {t("fields.dueDate")}
          <Input
            className="mt-1"
            disabled={busy}
            onChange={(event) => onChange({ ...form, dueDate: event.target.value })}
            type="date"
            value={form.dueDate}
          />
        </label>
        <label className="text-sm font-medium text-slate-700 md:col-span-2">
          {t("fields.assignee")}
          <div className="mt-1 grid gap-2 sm:grid-cols-[1fr_auto]">
            <Input
              disabled={busy}
              onChange={(event) => onChange({ ...form, assignedToUserId: event.target.value })}
              placeholder={t("fields.assigneePlaceholder")}
              value={form.assignedToUserId}
            />
            {canAssignToMe ? (
              <Button
                disabled={busy}
                onClick={() => onChange({ ...form, assignedToUserId: currentUserId ?? "" })}
                type="button"
                variant="secondary"
              >
                {t("assignMe")}
              </Button>
            ) : null}
          </div>
        </label>
        <label className="text-sm font-medium text-slate-700 md:col-span-2">
          {t("fields.description")}
          <textarea
            className="mt-1 min-h-24 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-950 outline-none transition placeholder:text-slate-400 focus:border-primary-600 focus:ring-2 focus:ring-primary-100 disabled:cursor-not-allowed disabled:bg-slate-100"
            disabled={busy}
            maxLength={500}
            onChange={(event) => onChange({ ...form, description: event.target.value })}
            value={form.description}
          />
        </label>
        <label className="text-sm font-medium text-slate-700 md:col-span-2">
          {t("fields.reason")}
          <Input
            className="mt-1"
            disabled={busy}
            maxLength={500}
            onChange={(event) => onChange({ ...form, reason: event.target.value })}
            value={form.reason}
          />
        </label>
      </div>
      <div className="mt-4 flex flex-wrap justify-end gap-2">
        <Button disabled={busy} onClick={onCancel} type="button" variant="secondary">
          {t("cancel")}
        </Button>
        <Button disabled={busy} type="submit">
          {busy ? t("saving") : editing ? t("saveChanges") : t("addItem")}
        </Button>
      </div>
    </form>
  );
}

function Badge({
  children,
  tone = "neutral"
}: {
  children: ReactNode;
  tone?: "neutral" | "warning" | "danger" | "ok";
}) {
  return (
    <span
      className={cn(
        "rounded-full px-2 py-0.5 text-xs font-medium",
        tone === "neutral" && "bg-slate-100 text-slate-700",
        tone === "warning" && "bg-amber-100 text-amber-800",
        tone === "danger" && "bg-red-100 text-red-800",
        tone === "ok" && "bg-emerald-100 text-emerald-800"
      )}
    >
      {children}
    </span>
  );
}

function compareChecklistItems(a: TripChecklistItem, b: TripChecklistItem) {
  if (a.sortOrder !== b.sortOrder) {
    return a.sortOrder - b.sortOrder;
  }
  return a.createdAt.localeCompare(b.createdAt);
}

function buildSummary(items: TripChecklistItem[], currentUserId?: string | null) {
  const checkedItems = items.filter((item) => item.checked).length;
  return {
    totalItems: items.length,
    checkedItems,
    uncheckedItems: items.length - checkedItems,
    highPriorityUnchecked: items.filter(
      (item) => !item.checked && (item.priority === "high" || item.priority === "critical")
    ).length,
    assignedToMe: items.filter((item) => currentUserId && item.assignedToUserId === currentUserId)
      .length,
    categories: []
  };
}

function createPayloadFromForm(form: ItemFormState): ChecklistItemPayload {
  return {
    title: form.title,
    description: form.description || null,
    category: form.category,
    itemType: form.itemType,
    priority: form.priority,
    quantity: parseOptionalInt(form.quantity),
    assignedToUserId: form.assignedToUserId || null,
    dueDate: form.dueDate || null,
    reason: form.reason || null
  };
}

function updatePayloadFromForm(form: ItemFormState): UpdateChecklistItemPayload {
  return {
    title: form.title,
    description: form.description || null,
    clearDescription: !form.description.trim(),
    category: form.category,
    itemType: form.itemType,
    priority: form.priority,
    quantity: parseOptionalInt(form.quantity),
    clearQuantity: !form.quantity.trim(),
    assignedToUserId: form.assignedToUserId || null,
    clearAssignee: !form.assignedToUserId.trim(),
    dueDate: form.dueDate || null,
    clearDueDate: !form.dueDate.trim(),
    reason: form.reason || null,
    clearReason: !form.reason.trim()
  };
}

function formFromItem(item: TripChecklistItem): ItemFormState {
  return {
    title: item.title,
    description: item.description ?? "",
    category: item.category,
    itemType: item.itemType,
    priority: item.priority,
    quantity: item.quantity != null ? String(item.quantity) : "",
    assignedToUserId: item.assignedToUserId ?? "",
    dueDate: item.dueDate ?? "",
    reason: item.reason ?? ""
  };
}

function parseOptionalInt(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }
  const parsed = Number.parseInt(trimmed, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
}

function createClientMutationId() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `offline-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function priorityTone(priority: ChecklistPriority) {
  if (priority === "critical") {
    return "danger";
  }
  if (priority === "high") {
    return "warning";
  }
  if (priority === "low") {
    return "ok";
  }
  return "neutral";
}
