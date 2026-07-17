"use client";

import { forwardRef } from "react";
import { cn } from "@/shared/lib/cn";

type SearchInputProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  label: string;
  className?: string;
};

export const SearchInput = forwardRef<HTMLInputElement, SearchInputProps>(
  function SearchInput({ value, onChange, placeholder, label, className }, ref) {
    return (
      <div className={cn("border-b border-slate-200 px-4 py-3", className)}>
        <label className="sr-only" htmlFor="global-command-palette-input">
          {label}
        </label>
        <input
          autoComplete="off"
          className="h-11 w-full bg-transparent text-[15px] text-slate-950 outline-none placeholder:text-slate-400"
          id="global-command-palette-input"
          onChange={(event) => onChange(event.target.value)}
          placeholder={placeholder}
          ref={ref}
          spellCheck={false}
          value={value}
        />
      </div>
    );
  }
);
