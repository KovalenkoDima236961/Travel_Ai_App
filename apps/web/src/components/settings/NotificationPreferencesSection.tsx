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
} from "@/types/notification-preferences";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";

const channels: Array<{ value: NotificationChannel; title: string }> = [
  { value: "in_app", title: "In-app notifications" },
  { value: "email", title: "Email notifications" }
];

const categories: Array<{
  value: NotificationCategory;
  label: string;
  description: string;
}> = [
  {
    value: "collaboration",
    label: "Collaboration invitations",
    description: "Invites and accepted collaboration requests."
  },
  {
    value: "comments",
    label: "Comments",
    description: "New comments on itinerary items."
  },
  {
    value: "role_changes",
    label: "Role changes",
    description: "When your collaborator role changes or you are removed from a trip."
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
    <Card>
      <div>
        <h2 className="text-lg font-semibold text-slate-950">Notification preferences</h2>
        <p className="mt-2 text-sm leading-6 text-slate-600">
          Choose how you want to be notified about trip activity.
        </p>
      </div>

      {preferencesQuery.isPending ? (
        <div className="mt-6 grid gap-5 md:grid-cols-2">
          <PreferenceSkeleton />
          <PreferenceSkeleton />
        </div>
      ) : null}

      {loadError ? (
        <div className="mt-6 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800" role="alert">
          {loadError}
        </div>
      ) : null}

      {!preferencesQuery.isPending && !loadError ? (
        <form
          className="mt-6 space-y-6"
          onSubmit={(event) => {
            event.preventDefault();
            preferencesMutation.mutate(orderedPreferences(draft));
          }}
        >
          <div className="grid gap-5 md:grid-cols-2">
            {channels.map((channel) => (
              <fieldset
                key={channel.value}
                className="rounded-md border border-slate-200 p-4"
                disabled={preferencesMutation.isPending}
              >
                <legend className="px-1 text-sm font-semibold text-slate-900">
                  {channel.title}
                </legend>
                <div className="mt-3 space-y-3">
                  {categories.map((category) => (
                    <label
                      key={category.value}
                      className="flex gap-3 rounded-md border border-slate-100 bg-slate-50 p-3"
                    >
                      <input
                        checked={checked(channel.value, category.value)}
                        className="mt-1 h-4 w-4 rounded border-slate-300 text-primary-600 focus:ring-primary-600"
                        type="checkbox"
                        onChange={(event) =>
                          toggle(channel.value, category.value, event.target.checked)
                        }
                      />
                      <span>
                        <span className="block text-sm font-medium text-slate-900">
                          {category.label}
                        </span>
                        <span className="mt-1 block text-sm leading-5 text-slate-600">
                          {category.description}
                        </span>
                      </span>
                    </label>
                  ))}
                </div>
              </fieldset>
            ))}
          </div>

          <p className="text-sm leading-6 text-slate-600">
            Disabling in-app collaboration notifications does not remove collaboration invitations from your Trips page.
          </p>

          {saveError ? (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800" role="alert">
              {saveError}
            </div>
          ) : null}

          {preferencesMutation.isSuccess ? (
            <div className="rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800" role="status">
              Notification preferences saved.
            </div>
          ) : null}

          <div className="flex justify-end">
            <Button disabled={preferencesMutation.isPending} type="submit">
              {preferencesMutation.isPending ? "Saving..." : "Save preferences"}
            </Button>
          </div>
        </form>
      ) : null}
    </Card>
  );
}

function PreferenceSkeleton() {
  return (
    <div className="rounded-md border border-slate-200 p-4">
      <div className="h-4 w-36 rounded bg-slate-200" />
      <div className="mt-4 space-y-3">
        {[0, 1, 2, 3].map((item) => (
          <div key={item} className="h-16 rounded-md bg-slate-100" />
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
