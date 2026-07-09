"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { cn } from "@/shared/lib/cn";
import { useAuth } from "@/components/auth/AuthProvider";
import { PushNotificationSettings } from "@/components/notifications/PushNotificationSettings";
import { NotificationPreferencesSection } from "@/components/settings/NotificationPreferencesSection";
import { PreferencesForm } from "@/components/settings/PreferencesForm";
import { ProfileForm } from "@/components/settings/ProfileForm";
import { PwaSettingsSection } from "@/components/settings/PwaSettingsSection";
import { SettingsSkeleton } from "@/components/settings/SettingsSkeleton";
import {
  getMyPreferences,
  getMyProfile,
  patchMyPreferences,
  updateMyProfile,
  userKeys
} from "@/lib/api/user";
import { getErrorMessage } from "@/lib/utils";
import { instrumentSans, newsreader } from "./fonts";
import { SettingsHeader } from "./SettingsHeader";

export function SettingsPageContent() {
  const queryClient = useQueryClient();
  const { user } = useAuth();

  const profileQuery = useQuery({
    queryKey: userKeys.profile(),
    queryFn: getMyProfile
  });

  const preferencesQuery = useQuery({
    queryKey: userKeys.preferences(),
    queryFn: getMyPreferences
  });

  const profileMutation = useMutation({
    mutationFn: updateMyProfile,
    onSuccess: async (profile) => {
      queryClient.setQueryData(userKeys.profile(), profile);
      await queryClient.invalidateQueries({ queryKey: userKeys.profile() });
    }
  });

  const preferencesMutation = useMutation({
    mutationFn: patchMyPreferences,
    onSuccess: async (preferences) => {
      queryClient.setQueryData(userKeys.preferences(), preferences);
      await queryClient.invalidateQueries({ queryKey: userKeys.preferences() });
    }
  });

  const isLoading = profileQuery.isPending || preferencesQuery.isPending;
  const loadError = profileQuery.error ?? preferencesQuery.error;

  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <SettingsHeader />

      {/* Content region is a div, not <main> — the root layout already provides
          the <main> landmark, and nesting a second one is invalid. */}
      <div className="mx-auto max-w-[960px] px-6 pb-[72px] pt-12 sm:px-10">
        <div className="max-w-[640px]">
          <h1 className="font-newsreader text-[44px] font-medium leading-[1.05] tracking-[-0.02em] text-cocoa-900">
            Profile &amp; preferences
          </h1>
          <p className="mt-3.5 text-[16px] leading-relaxed text-cocoa-500">
            The details used to personalize your AI-generated itineraries.
          </p>
        </div>

        {isLoading ? <div className="mt-9"><SettingsSkeleton /></div> : null}

        {loadError ? (
          <div
            className="mt-9 rounded-[20px] border border-clay/30 bg-clay-tint/50 p-7 text-[14.5px] text-clay-deep"
            role="alert"
          >
            {getErrorMessage(
              loadError,
              "Could not load settings. Confirm User Service is running."
            )}
          </div>
        ) : null}

        {profileQuery.data && preferencesQuery.data ? (
          <div className="mt-9 flex flex-col gap-5">
            <ProfileForm
              email={user?.email ?? null}
              errorMessage={
                profileMutation.isError
                  ? getErrorMessage(profileMutation.error, "Could not save profile.")
                  : null
              }
              isSaving={profileMutation.isPending}
              profile={profileQuery.data}
              successMessage={profileMutation.isSuccess ? "Profile saved." : null}
              onSubmit={(values) => profileMutation.mutate(values)}
            />

            <PreferencesForm
              errorMessage={
                preferencesMutation.isError
                  ? getErrorMessage(preferencesMutation.error, "Could not save preferences.")
                  : null
              }
              isSaving={preferencesMutation.isPending}
              preferences={preferencesQuery.data}
              successMessage={preferencesMutation.isSuccess ? "Preferences saved." : null}
              onSubmit={(values) => preferencesMutation.mutate(values)}
            />

            <PwaSettingsSection />

            <div id="push-notifications">
              <PushNotificationSettings />
            </div>

            <NotificationPreferencesSection />
          </div>
        ) : null}
      </div>
    </div>
  );
}
