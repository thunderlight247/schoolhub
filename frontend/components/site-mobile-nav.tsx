"use client";

// Mobile nav for the marketing homepage. The desktop header hides its
// section links below `lg` — on a phone that left only an "Apply now"
// button, so mobile visitors had no in-page (or portal) navigation. This
// renders a hamburger that expands into a tap-friendly list of the same
// links plus the portal sign-in. Client state is needed because the menu
// must open/close on tap; the rest of the homepage stays a static server
// component.

import { useState } from "react";
import Link from "next/link";

const LINKS = [
  { href: "#about", label: "About" },
  { href: "#academics", label: "Academics" },
  { href: "#admissions", label: "Admissions" },
  { href: "#news", label: "News" },
  { href: "#contact", label: "Contact" },
];

export function SiteMobileNav() {
  const [open, setOpen] = useState(false);

  return (
    <div className="lg:hidden">
      <button
        type="button"
        aria-expanded={open}
        aria-controls="mobile-menu"
        aria-label={open ? "Close menu" : "Open menu"}
        onClick={() => setOpen(!open)}
        className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-700 transition-colors hover:bg-slate-100"
      >
        <span aria-hidden className="text-2xl leading-none">
          {open ? "✕" : "☰"}
        </span>
      </button>

      {open && (
        <div id="mobile-menu" className="absolute inset-x-0 top-full z-40 border-b border-slate-200 bg-white shadow-lg">
          <nav aria-label="Mobile" className="mx-auto max-w-6xl px-6 py-4">
            <ul className="divide-y divide-slate-100">
              {LINKS.map((l) => (
                <li key={l.href}>
                  <a
                    href={l.href}
                    onClick={() => setOpen(false)}
                    className="flex min-h-11 items-center px-1 py-2.5 text-sm font-medium text-slate-700"
                  >
                    {l.label}
                  </a>
                </li>
              ))}
              <li>
                <Link
                  href="/login"
                  onClick={() => setOpen(false)}
                  className="flex min-h-11 items-center px-1 py-2.5 text-sm font-semibold text-brand-700"
                >
                  Portal sign-in
                </Link>
              </li>
            </ul>
          </nav>
        </div>
      )}
    </div>
  );
}
