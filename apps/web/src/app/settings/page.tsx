"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { PushNotificationSettings } from "@/components/notifications/PushNotificationSettings";
import { NotificationPreferencesSection } from "@/components/settings/NotificationPreferencesSection";
import { PreferencesForm } from "@/components/settings/PreferencesForm";
import { ProfileForm } from "@/components/settings/ProfileForm";
import { SettingsSkeleton } from "@/components/settings/SettingsSkeleton";
import {
  getMyPreferences,
  getMyProfile,
  patchMyPreferences,
  updateMyProfile,
  userKeys
} from "@/lib/api/user";
import { getErrorMessage } from "@/lib/utils";

export default function SettingsPage() {
  return (
    <ProtectedRoute>
      <SettingsPageContent />
    </ProtectedRoute>
  );
}

function SettingsPageContent() {
  const queryClient = useQueryClient();

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
    <PageContainer className="max-w-5xl">
      <div className="mb-8">
        <p className="text-sm font-semibold uppercase text-primary-700">Settings</p>
        <h1 className="mt-2 text-3xl font-semibold text-slate-950">Profile and preferences</h1>
        <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-600">
          Manage the details used to personalize new itinerary generations.
        </p>
      </div>

      {isLoading ? <SettingsSkeleton /> : null}

      {loadError ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800" role="alert">
          {getErrorMessage(
            loadError,
            "Could not load settings. Confirm User Service is running."
          )}
        </div>
      ) : null}

      {profileQuery.data && preferencesQuery.data ? (
        <div className="space-y-6">
          <ProfileForm
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

          <PushNotificationSettings />

          <NotificationPreferencesSection />
        </div>
      ) : null}
    </PageContainer>
  );
}
