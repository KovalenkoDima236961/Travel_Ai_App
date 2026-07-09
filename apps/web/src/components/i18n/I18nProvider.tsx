"use client";

import { useQuery } from "@tanstack/react-query";
import { NextIntlClientProvider, type AbstractIntlMessages } from "next-intl";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode
} from "react";
import { useAuth } from "@/components/auth/AuthProvider";
import { getMyProfile, userKeys } from "@/lib/api/user";
import {
  DEFAULT_LANGUAGE,
  getInitialLanguage,
  normalizeLanguage,
  setStoredLanguage,
  type SupportedLanguage
} from "@/lib/i18n/languages";
import en from "../../../messages/en.json";
import es from "../../../messages/es.json";
import uk from "../../../messages/uk.json";
import fr from "../../../messages/fr.json";

const CATALOGS: Record<SupportedLanguage, AbstractIntlMessages> = { en, es, uk, fr };

type I18nContextValue = {
  language: SupportedLanguage;
  setLanguage: (language: SupportedLanguage) => void;
};

const I18nContext = createContext<I18nContextValue | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated, user } = useAuth();
  const [language, setLanguageState] = useState<SupportedLanguage>(DEFAULT_LANGUAGE);
  const profileQuery = useQuery({
    queryKey: userKeys.profile(),
    queryFn: getMyProfile,
    enabled: isAuthenticated && Boolean(user?.id),
    staleTime: 60_000
  });

  const setLanguage = useCallback((nextLanguage: SupportedLanguage) => {
    setLanguageState(nextLanguage);
    setStoredLanguage(nextLanguage);
    document.documentElement.lang = nextLanguage;
  }, []);

  useEffect(() => {
    setLanguage(getInitialLanguage());
  }, [setLanguage]);

  useEffect(() => {
    if (!profileQuery.data?.preferredLanguage) {
      return;
    }
    const profileLanguage = normalizeLanguage(profileQuery.data.preferredLanguage);
    setLanguage(profileLanguage);
  }, [profileQuery.data?.preferredLanguage, setLanguage]);

  const value = useMemo(() => ({ language, setLanguage }), [language, setLanguage]);
  const messages = useMemo(
    () => mergeMessages(en as AbstractIntlMessages, CATALOGS[language]),
    [language]
  );

  return (
    <I18nContext.Provider value={value}>
      <NextIntlClientProvider
        locale={language}
        messages={messages}
        timeZone="Europe/Bratislava"
        onError={(error) => {
          if (error.code !== "MISSING_MESSAGE") {
            console.error(error);
          }
        }}
      >
        {children}
      </NextIntlClientProvider>
    </I18nContext.Provider>
  );
}

export function useAppLanguage(): I18nContextValue {
  const value = useContext(I18nContext);
  if (!value) {
    throw new Error("useAppLanguage must be used within I18nProvider");
  }
  return value;
}

function mergeMessages(
  fallback: AbstractIntlMessages,
  selected: AbstractIntlMessages
): AbstractIntlMessages {
  const result: AbstractIntlMessages = { ...fallback };
  for (const [key, value] of Object.entries(selected)) {
    const fallbackValue = result[key];
    if (isMessageObject(fallbackValue) && isMessageObject(value)) {
      result[key] = mergeMessages(fallbackValue, value);
    } else {
      result[key] = value;
    }
  }
  return result;
}

function isMessageObject(value: unknown): value is AbstractIntlMessages {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
