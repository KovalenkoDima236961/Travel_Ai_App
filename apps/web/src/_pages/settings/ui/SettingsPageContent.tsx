"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { ErrorState } from "@/components/ui";
import { cn } from "@/shared/lib/cn";
import { useAuth } from "@/components/auth/AuthProvider";
import { PushNotificationSettings } from "@/components/notifications/PushNotificationSettings";
import { NotificationPreferencesSection } from "@/components/settings/NotificationPreferencesSection";
import { OnboardingSettingsCard } from "@/components/settings/OnboardingSettingsCard";
import { PreferencesForm } from "@/components/settings/PreferencesForm";
import { ProfileForm } from "@/components/settings/ProfileForm";
import { PwaSettingsSection } from "@/components/settings/PwaSettingsSection";
import { DataPrivacySettings } from "@/components/data";
import { SettingsSkeleton } from "@/components/settings/SettingsSkeleton";
import { LanguageSelector } from "@/components/i18n/LanguageSelector";
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
import { PreferenceCompletenessCard } from "@/components/personalization";
import { usePreferenceCompleteness } from "@/hooks/usePersonalization";

export function SettingsPageContent() {
  const translate = useTranslations("settings");
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
  const completenessQuery = usePreferenceCompleteness();

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
      <div className="mx-auto max-w-[960px] px-4 pb-[72px] pt-8 sm:px-10 sm:pt-12">
        <div className="max-w-[640px]">
          <h1 className="font-newsreader text-[44px] font-medium leading-[1.05] tracking-[-0.02em] text-cocoa-900">
            {translate("title")}
          </h1>
          <p className="mt-3.5 text-[16px] leading-relaxed text-cocoa-500">
            {translate("description")}
          </p>
        </div>

        {isLoading ? <div className="mt-9"><SettingsSkeleton /></div> : null}

        {loadError ? (
          <ErrorState
            className="mt-9 rounded-[20px]"
            description={translate("loadFailed")}
            developmentDetails={loadError instanceof Error ? loadError.message : undefined}
            retryAction={{
              onRetry: () => {
                void profileQuery.refetch();
                void preferencesQuery.refetch();
              },
              pending: profileQuery.isFetching || preferencesQuery.isFetching
            }}
            title={translate("loadErrorTitle")}
          />
        ) : null}

        {profileQuery.data && preferencesQuery.data ? (
          <div className="mt-9 flex flex-col gap-5">
            {completenessQuery.data ? <PreferenceCompletenessCard value={completenessQuery.data} /> : null}
            <section className="rounded-[20px] border border-sand-300 bg-white p-7">
              <h2 className="font-newsreader text-2xl font-medium text-cocoa-900">
                {translate("language")}
              </h2>
              <p className="mb-5 mt-2 text-sm leading-relaxed text-cocoa-500">
                {translate("languageDescription")}
              </p>
              <LanguageSelector />
            </section>

            <ProfileForm
              email={user?.email ?? null}
              errorMessage={
                profileMutation.isError
                  ? getErrorMessage(profileMutation.error, translate("saveFailed"))
                  : null
              }
              isSaving={profileMutation.isPending}
              profile={profileQuery.data}
              successMessage={profileMutation.isSuccess ? translate("profileSaved") : null}
              onSubmit={(values) => profileMutation.mutate(values)}
            />

            <PreferencesForm
              errorMessage={
                preferencesMutation.isError
                  ? translate("preferencesSaveFailed")
                  : null
              }
              isSaving={preferencesMutation.isPending}
              preferences={preferencesQuery.data}
              successMessage={preferencesMutation.isSuccess ? translate("preferencesSaved") : null}
              onSubmit={(values) => preferencesMutation.mutate(values)}
            />

            <OnboardingSettingsCard />

            <PwaSettingsSection />

            <DataPrivacySettings />

            <div id="push-notifications">
              <PushNotificationSettings />
            </div>

            <div id="notifications">
              <NotificationPreferencesSection />
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}
