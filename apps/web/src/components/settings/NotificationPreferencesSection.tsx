"use client";

import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getNotificationPreferences,
  notificationPreferenceKeys,
  updateNotificationPreferences
} from "@/lib/api/notification-preferences";
import { getErrorMessage } from "@/lib/utils";
import type {
  NotificationCategory,
  NotificationChannel,
  NotificationPreference
} from "@/entities/notification-preferences/model";
import {
  PrimaryButton,
  SaveNotice,
  SectionHeading,
  SettingsCard,
  Switch
} from "@/components/settings/controls";

const channels: Array<{ value: NotificationChannel; title: string }> = [
  { value: "in_app", title: "In-app notifications" },
  { value: "email", title: "Email notifications" },
  { value: "push", title: "Push notifications" }
];

const categories: Array<{
  value: NotificationCategory;
  label: string;
  description: string;
}> = [
  {
    value: "collaboration",
    label: "Collaboration invitations",
    description: "Trip and workspace invites plus accepted requests."
  },
  {
    value: "comments",
    label: "Comments",
    description: "New comments on itinerary items."
  },
  {
    value: "role_changes",
    label: "Role changes",
    description: "Trip collaborator and workspace role changes or removals."
  },
  {
    value: "trip_updates",
    label: "Trip updates",
    description: "Itinerary edits, regenerations, and restored versions."
  }
];

export function NotificationPreferencesSection() {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<NotificationPreference[]>([]);

  const preferencesQuery = useQuery({
    queryKey: notificationPreferenceKeys.all,
    queryFn: getNotificationPreferences
  });

  useEffect(() => {
    if (preferencesQuery.data) {
      setDraft(preferencesQuery.data);
    }
  }, [preferencesQuery.data]);

  const preferencesMutation = useMutation({
    mutationFn: updateNotificationPreferences,
    onSuccess: async (items) => {
      setDraft(items);
      queryClient.setQueryData(notificationPreferenceKeys.all, items);
      await queryClient.invalidateQueries({ queryKey: notificationPreferenceKeys.all });
    }
  });

  const byKey = useMemo(() => {
    const map = new Map<string, NotificationPreference>();
    for (const item of draft) {
      map.set(preferenceKey(item.channel, item.category), item);
    }
    return map;
  }, [draft]);

  function checked(channel: NotificationChannel, category: NotificationCategory) {
    return byKey.get(preferenceKey(channel, category))?.enabled ?? false;
  }

  function toggle(
    channel: NotificationChannel,
    category: NotificationCategory,
    enabled: boolean
  ) {
    setDraft((items) => {
      const key = preferenceKey(channel, category);
      let found = false;
      const next = items.map((item) => {
        if (preferenceKey(item.channel, item.category) !== key) {
          return item;
        }
        found = true;
        return { ...item, enabled };
      });
      if (!found) {
        next.push({ channel, category, enabled });
      }
      return next;
    });
  }

  const loadError = preferencesQuery.error
    ? getErrorMessage(
        preferencesQuery.error,
        "Could not load notification preferences. Confirm Notification Service is running."
      )
    : null;
  const saveError = preferencesMutation.isError
    ? getErrorMessage(preferencesMutation.error, "Could not save notification preferences.")
    : null;

  return (
    <SettingsCard>
      <SectionHeading
        title="Notification preferences"
        subtitle="Choose how you want to be notified about trip activity."
      />

      {preferencesQuery.isPending ? (
        <div className="mt-6 grid gap-4 lg:grid-cols-3">
          <PreferenceSkeleton />
          <PreferenceSkeleton />
          <PreferenceSkeleton />
        </div>
      ) : null}

      {loadError ? (
        <div className="mt-6">
          <SaveNotice errorMessage={loadError} />
        </div>
      ) : null}

      {!preferencesQuery.isPending && !loadError ? (
        <form
          className="mt-6 flex flex-col gap-5"
          onSubmit={(event) => {
            event.preventDefault();
            preferencesMutation.mutate(orderedPreferences(draft));
          }}
        >
          <div className="grid gap-4 lg:grid-cols-3">
            {channels.map((channel) => (
              <fieldset
                key={channel.value}
                className="rounded-2xl border border-sand-300 bg-sand-50/60 p-4"
                disabled={preferencesMutation.isPending}
              >
                <legend className="px-1 text-[13.5px] font-semibold text-cocoa-900">
                  {channel.title}
                </legend>
                <div className="mt-2 flex flex-col">
                  {categories.map((category, index) => (
                    <div
                      key={category.value}
                      className={`flex items-start justify-between gap-3 py-3 ${
                        index > 0 ? "border-t border-sand-200" : ""
                      }`}
                    >
                      <div className="min-w-0">
                        <p className="text-[13.5px] font-semibold text-cocoa-900">
                          {category.label}
                        </p>
                        <p className="mt-0.5 text-[12.5px] leading-[1.5] text-cocoa-400">
                          {category.description}
                        </p>
                      </div>
                      <Switch
                        label={`${channel.title}: ${category.label}`}
                        checked={checked(channel.value, category.value)}
                        disabled={preferencesMutation.isPending}
                        onChange={(next) => toggle(channel.value, category.value, next)}
                      />
                    </div>
                  ))}
                </div>
              </fieldset>
            ))}
          </div>

          <p className="text-[13.5px] leading-relaxed text-cocoa-500">
            Disabling in-app collaboration notifications does not remove collaboration invitations
            from your Trips page.
          </p>

          {saveError ? <SaveNotice errorMessage={saveError} /> : null}

          {preferencesMutation.isSuccess ? (
            <SaveNotice successMessage="Notification preferences saved." />
          ) : null}

          <div className="flex justify-end">
            <PrimaryButton disabled={preferencesMutation.isPending} type="submit">
              {preferencesMutation.isPending ? "Saving…" : "Save preferences"}
            </PrimaryButton>
          </div>
        </form>
      ) : null}
    </SettingsCard>
  );
}

function PreferenceSkeleton() {
  return (
    <div className="rounded-2xl border border-sand-300 p-4">
      <div className="h-4 w-36 rounded bg-sand-300" />
      <div className="mt-4 space-y-3">
        {[0, 1, 2, 3].map((item) => (
          <div key={item} className="h-14 rounded-xl bg-sand-200" />
        ))}
      </div>
    </div>
  );
}

function orderedPreferences(items: NotificationPreference[]) {
  const map = new Map<string, NotificationPreference>();
  for (const item of items) {
    map.set(preferenceKey(item.channel, item.category), item);
  }
  return channels.flatMap((channel) =>
    categories.map((category) => {
      const current = map.get(preferenceKey(channel.value, category.value));
      return {
        channel: channel.value,
        category: category.value,
        enabled: current?.enabled ?? false
      };
    })
  );
}

function preferenceKey(channel: NotificationChannel, category: NotificationCategory) {
  return `${channel}:${category}`;
}
