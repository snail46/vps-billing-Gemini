import React from "react";
import { useI18n } from "../i18n";

interface LanguageSwitcherProps {
  className?: string;
}

export const LanguageSwitcher: React.FC<LanguageSwitcherProps> = ({ className = "" }) => {
  const { locale, setLocale, t } = useI18n();

  return (
    <div className={`inline-flex items-center gap-1.5 text-xs font-medium ${className}`}>
      <span className="text-zinc-400">{t("common.language")}:</span>
      <div className="inline-flex rounded-lg border border-zinc-200 p-0.5 bg-zinc-50">
        <button
          type="button"
          onClick={() => setLocale("zh-CN")}
          className={`px-2 py-1 rounded-md text-xs font-medium transition-colors ${
            locale === "zh-CN"
              ? "bg-white text-zinc-900 shadow-sm"
              : "text-zinc-500 hover:text-zinc-900"
          }`}
        >
          中文
        </button>
        <button
          type="button"
          onClick={() => setLocale("en-US")}
          className={`px-2 py-1 rounded-md text-xs font-medium transition-colors ${
            locale === "en-US"
              ? "bg-white text-zinc-900 shadow-sm"
              : "text-zinc-500 hover:text-zinc-900"
          }`}
        >
          English
        </button>
      </div>
    </div>
  );
};
