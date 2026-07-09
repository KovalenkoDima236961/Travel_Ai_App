"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { useAuth } from "@/components/auth/AuthProvider";
import { useAppLanguage } from "@/components/i18n/I18nProvider";
import { getMyProfile, updateMyProfile, userKeys } from "@/lib/api/user";
import {
  LANGUAGE_LABELS,
  SUPPORTED_LANGUAGES,
  type SupportedLanguage
} from "@/lib/i18n/languages";

export function LanguageSelector({ compact = false }: { compact?: boolean }) {
  const translate = useTranslations("settings");
  const { isAuthenticated } = useAuth();
  const { language, setLanguage } = useAppLanguage();
  const queryClient = useQueryClient();
  const profileQuery = useQuery({
    queryKey: userKeys.profile(),
    queryFn: getMyProfile,
    enabled: isAuthenticated
  });
  const mutation = useMutation({
    mutationFn: updateMyProfile,
    onSuccess: (profile) => {
      queryClient.setQueryData(userKeys.profile(), profile);
    }
  });

  function handleChange(nextLanguage: SupportedLanguage) {
    setLanguage(nextLanguage);
    const profile = profileQuery.data;
    if (isAuthenticated && profile) {
      mutation.mutate({ ...profile, preferredLanguage: nextLanguage });
    }
  }

  return (
    <div className={compact ? "min-w-32" : "space-y-2"}>
      {!compact ? (
        <label htmlFor="app-language" className="block text-sm font-semibold text-cocoa-800">
          {translate("language")}
        </label>
      ) : null}
      <select
        id={compact ? "app-language-compact" : "app-language"}
        aria-label={translate("language")}
        value={language}
        onChange={(event) => handleChange(event.target.value as SupportedLanguage)}
        disabled={mutation.isPending}
        className="w-full rounded-xl border border-sand-300 bg-white px-3 py-2 text-sm text-cocoa-800 outline-none focus:border-[#3E6B5A] focus:ring-2 focus:ring-[#3E6B5A]/20"
      >
        {SUPPORTED_LANGUAGES.map((code) => (
          <option key={code} value={code}>
            {LANGUAGE_LABELS[code]}
          </option>
        ))}
      </select>
      {!compact && mutation.isSuccess ? (
        <p className="text-sm text-[#3E6B5A]" role="status">
          {translate("languageUpdated")}
        </p>
      ) : null}
      {!compact && mutation.isError ? (
        <p className="text-sm text-clay-deep" role="alert">
          {translate("languageUpdateFailed")}
        </p>
      ) : null}
    </div>
  );
}
