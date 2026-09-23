import React, { createContext, useContext, useState, useEffect } from "react";
import i18n from "i18next";
import { initReactI18next, useTranslation } from "react-i18next";
import zhCN from "./locales/zh-CN.json";
import enUS from "./locales/en-US.json";

export type Locale = "zh-CN" | "en-US";

if (!i18n.isInitialized) {
  i18n.use(initReactI18next).init({
    resources: {
      "zh-CN": { translation: zhCN },
      "en-US": { translation: enUS },
    },
    lng: "zh-CN",
    fallbackLng: "zh-CN",
    interpolation: {
      escapeValue: false,
    },
  });
}

interface I18nContextType {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: string, params?: Record<string, string | number>) => string;
}

const I18nContext = createContext<I18nContextType>({
  locale: "zh-CN",
  setLocale: () => {},
  t: (key: string) => key,
});

export const I18nProvider: React.FC<{
  children: React.ReactNode;
  defaultLocale?: Locale;
  storageKey?: string;
}> = ({ children, defaultLocale = "zh-CN", storageKey = "preferred_locale" }) => {
  const { t: i18nextT } = useTranslation();
  const [locale, setLocaleState] = useState<Locale>(() => {
    if (typeof window !== "undefined") {
      const saved = localStorage.getItem(storageKey);
      if (saved === "zh-CN" || saved === "en-US") {
        return saved;
      }
      if (navigator.language.startsWith("zh")) {
        return "zh-CN";
      }
    }
    return defaultLocale;
  });

  useEffect(() => {
    i18n.changeLanguage(locale);
  }, [locale]);

  const setLocale = (newLocale: Locale) => {
    setLocaleState(newLocale);
    if (typeof window !== "undefined") {
      localStorage.setItem(storageKey, newLocale);
    }
  };

  const t = (key: string, params?: Record<string, string | number>): string => {
    return i18nextT(key, params || {}) as string;
  };

  return (
    <I18nContext.Provider value={{ locale, setLocale, t }}>
      {children}
    </I18nContext.Provider>
  );
};

export const useI18n = () => useContext(I18nContext);
export { i18n, useTranslation };
