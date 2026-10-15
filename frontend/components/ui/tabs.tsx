"use client";

// Tabs (M33): shared accessible tab bar. Migrates every hand-rolled tab row
// (reports, academics workspace, all three portals) onto proper ARIA tabs:
//
//   <Tabs active={tab} onChange={setTab} variant="underline" label="Report types"
//         tabs={[{ key: "enrollment", label: "Enrollment Summary" }, ...]} />
//
// Keyboard support follows the WAI-ARIA convention: Left/Right arrows move
// selection (wrap), Home/End jump to the ends, and focus follows selection
// (roving tabindex — the active tab is the only one tabbed to).

import { useId, useRef, type KeyboardEvent as ReactKeyboardEvent } from "react";

export interface TabItem<T extends string> {
  key: T;
  label: string;
  icon?: string;
}

interface TabsProps<T extends string> {
  items: readonly TabItem<T>[];
  active: T;
  onChange: (key: T) => void;
  /** "underline" = classic content tabs; "pills" = segmented-control look. */
  variant?: "underline" | "pills";
  /** Accessible name for the tablist (there may be more than one tab group per page). */
  label: string;
  className?: string;
}

export function Tabs<T extends string>({
  items,
  active,
  onChange,
  variant = "underline",
  label,
  className = "",
}: TabsProps<T>) {
  const uid = useId();
  const listRef = useRef<HTMLDivElement>(null);

  const move = (dir: 1 | -1 | "first" | "last") => {
    const i = items.findIndex((t) => t.key === active);
    let next: number;
    if (dir === "first") next = 0;
    else if (dir === "last") next = items.length - 1;
    else next = (i + dir + items.length) % items.length;
    const target = items[next];
    if (target && target.key !== active) onChange(target.key);
    // Focus follows selection; the selected button re-renders with tabIndex 0.
    requestAnimationFrame(() => {
      const el = listRef.current?.querySelector<HTMLButtonElement>(
        `#${CSS.escape(`${uid}-tab-${next}`)}`,
      );
      el?.focus();
    });
  };

  const onKeyDown = (e: ReactKeyboardEvent) => {
    if (e.key === "ArrowRight") {
      e.preventDefault();
      move(1);
    } else if (e.key === "ArrowLeft") {
      e.preventDefault();
      move(-1);
    } else if (e.key === "Home") {
      e.preventDefault();
      move("first");
    } else if (e.key === "End") {
      e.preventDefault();
      move("last");
    }
  };

  const common = (item: TabItem<T>, i: number) => ({
    id: `${uid}-tab-${i}`,
    role: "tab" as const,
    "aria-selected": item.key === active,
    tabIndex: item.key === active ? 0 : -1,
  });

  if (variant === "pills") {
    return (
      <div
        ref={listRef}
        role="tablist"
        aria-label={label}
        onKeyDown={onKeyDown}
        className={`flex gap-1 overflow-x-auto rounded-lg border border-slate-200 bg-white p-1 shadow-sm ${className}`}
      >
        {items.map((item, i) => {
          const on = item.key === active;
          return (
            <button
              key={item.key}
              {...common(item, i)}
              onClick={() => onChange(item.key)}
              className={`whitespace-nowrap rounded-md px-4 py-1.5 text-sm font-medium ${
                on ? "bg-brand-50 text-brand-900" : "text-slate-600 hover:bg-slate-50"
              }`}
            >
              {item.icon && <span aria-hidden className="mr-1">{item.icon}</span>}
              {item.label}
            </button>
          );
        })}
      </div>
    );
  }

  return (
    <div
      ref={listRef}
      role="tablist"
      aria-label={label}
      onKeyDown={onKeyDown}
      className={`flex overflow-x-auto border-b border-slate-200 text-sm font-medium ${className}`}
    >
      {items.map((item, i) => {
        const on = item.key === active;
        return (
          <button
            key={item.key}
            {...common(item, i)}
            onClick={() => onChange(item.key)}
            className={`pb-3 px-4 border-b-2 whitespace-nowrap transition-colors ${
              on
                ? "border-brand-600 text-brand-700 font-semibold"
                : "border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300"
            }`}
          >
            {item.icon && <span aria-hidden className="mr-1">{item.icon}</span>}
            {item.label}
          </button>
        );
      })}
    </div>
  );
}
