"use client";

// SearchInput (M23): a debounced text search for list screens. Fires
// onSearch after `debounceMs` of typing inactivity (default 300ms) so list
// endpoints aren't hit on every keystroke, and reports a clearing keystroke
// immediately (empty string never waits on the timer).

import { useEffect, useRef, useState } from "react";

interface SearchInputProps {
  placeholder?: string;
  onSearch: (query: string) => void;
  debounceMs?: number;
  className?: string;
}

export function SearchInput({
  placeholder = "Search…",
  onSearch,
  debounceMs = 300,
  className = "",
}: SearchInputProps) {
  const [value, setValue] = useState("");
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onSearchRef = useRef(onSearch);
  onSearchRef.current = onSearch;

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  const handleChange = (next: string) => {
    setValue(next);
    if (timerRef.current) clearTimeout(timerRef.current);
    if (next === "") {
      onSearchRef.current("");
      return;
    }
    timerRef.current = setTimeout(() => onSearchRef.current(next), debounceMs);
  };

  return (
    <div
      className={`flex w-full items-center gap-2 rounded-full bg-white px-3 ring-[1.5px] ring-slate-300 shadow-sm transition-shadow focus-within:ring-brand-600 ${className}`}
    >
      {/* Placeholder icon until a final icon set is chosen; swap the img src
          when the design system settles on real assets. */}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img src="/icons/search.png" alt="" width={14} height={14} className="opacity-60" />
      <input
        type="search"
        value={value}
        onChange={(e) => handleChange(e.target.value)}
        placeholder={placeholder}
        aria-label={placeholder}
        className="w-full bg-transparent p-2 text-sm text-slate-900 outline-none placeholder:text-slate-400"
      />
    </div>
  );
}
