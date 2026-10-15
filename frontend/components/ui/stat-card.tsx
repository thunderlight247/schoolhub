"use client";

// Small presentation primitives (M25) following the professional dashboard
// reference: pastel stat cards with a context chip, and initial-avatars for
// people rows (no photo assets yet — initials render identically at any
// scale and need no network requests).

const STAT_TONES = ["purple", "sky", "yellow", "green"] as const;
export type StatTone = (typeof STAT_TONES)[number];

const toneClasses: Record<StatTone, string> = {
  purple: "bg-violet-50",
  sky: "bg-sky-50",
  yellow: "bg-amber-50",
  green: "bg-emerald-50",
};

export function StatCard({
  label,
  value,
  chip,
  tone = "sky",
}: {
  label: string;
  value: number | string;
  /** Small context chip in the corner, e.g. the academic year. */
  chip?: string;
  tone?: StatTone;
}) {
  return (
    <div
      className={`flex-1 rounded-2xl p-4 shadow-sm min-w-[130px] ${toneClasses[tone]}`}
    >
      <div className="flex items-center justify-between">
        {chip ? (
          <span className="rounded-full bg-white px-2 py-1 text-[10px] font-medium text-emerald-600">
            {chip}
          </span>
        ) : (
          <span />
        )}
      </div>
      <h3 className="my-3 text-2xl font-semibold text-slate-900">{value}</h3>
      <h4 className="text-sm font-medium capitalize text-slate-500">{label}</h4>
    </div>
  );
}

const AVATAR_TONES = [
  "bg-brand-100 text-brand-800",
  "bg-sky-100 text-sky-800",
  "bg-violet-100 text-violet-800",
  "bg-amber-100 text-amber-800",
  "bg-emerald-100 text-emerald-800",
  "bg-rose-100 text-rose-800",
];

export function Avatar({
  firstName,
  lastName,
  size = "md",
}: {
  firstName: string;
  lastName: string;
  size?: "sm" | "md" | "lg";
}) {
  const initials = `${firstName.charAt(0)}${lastName.charAt(0)}`.toUpperCase();
  // Stable tone per person so rows don't reshuffle colors on refetch.
  const tone =
    AVATAR_TONES[
      Math.abs(
        [...`${firstName}${lastName}`].reduce((a, c) => a * 31 + c.charCodeAt(0), 7),
      ) % AVATAR_TONES.length
    ];
  const dims = { sm: "h-8 w-8 text-xs", md: "h-10 w-10 text-sm", lg: "h-14 w-14 text-lg" }[size];
  return (
    <span
      aria-hidden
      className={`inline-flex shrink-0 items-center justify-center rounded-full font-semibold ${dims} ${tone}`}
    >
      {initials}
    </span>
  );
}

/** The reference-repo "Info" first cell: avatar with name + subtitle stacked. */
export function PersonCell({
  firstName,
  lastName,
  subtitle,
}: {
  firstName: string;
  lastName: string;
  subtitle?: string;
}) {
  return (
    <div className="flex items-center gap-3">
      <Avatar firstName={firstName} lastName={lastName} />
      <div className="flex min-w-0 flex-col">
        <span className="truncate font-semibold text-slate-800">
          {firstName} {lastName}
        </span>
        {subtitle && <span className="truncate text-xs text-slate-500">{subtitle}</span>}
      </div>
    </div>
  );
}
