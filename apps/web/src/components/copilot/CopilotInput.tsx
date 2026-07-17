"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";

export function CopilotInput({ disabled, onSend }: { disabled?: boolean; onSend: (message: string) => void }) {
  const t = useTranslations("copilot");
  const [value, setValue] = useState("");
  const submit = () => {
    if (!value.trim() || disabled) {
      return;
    }
    onSend(value);
    setValue("");
  };
  return (
    <form
      className="flex items-end gap-2 border-t border-sand-300 bg-white p-3"
      onSubmit={(event) => {
        event.preventDefault();
        submit();
      }}
    >
      <label className="sr-only" htmlFor="copilot-message">{t("ask")}</label>
      <textarea
        className="min-h-11 flex-1 resize-none rounded-xl border border-sand-400 bg-sand-50 px-3 py-2 text-sm text-cocoa-700 outline-none placeholder:text-cocoa-400 focus:border-cocoa-500 focus:ring-2 focus:ring-primary-600"
        disabled={disabled}
        id="copilot-message"
        maxLength={2000}
        onChange={(event) => setValue(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter" && !event.shiftKey) {
            event.preventDefault();
            submit();
          }
        }}
        placeholder={t("askPlaceholder")}
        rows={1}
        value={value}
      />
      <Button disabled={disabled || !value.trim()} size="sm" type="submit">{t("send")}</Button>
    </form>
  );
}
