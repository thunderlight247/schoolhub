"use client";

// Form primitives (M23): small, unopinionated building blocks so the create/
// edit screens of M24+ look and behave identically. Deliberately NOT a form
// library — plain controlled inputs with shared styling and a conventional
// error/success feedback row.

import { useId, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes, type TextareaHTMLAttributes } from "react";

const baseField =
  "mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm placeholder:text-slate-400 focus:border-brand-600 focus:outline-none focus:ring-1 focus:ring-brand-600 disabled:bg-slate-50";

const invalidField = `${baseField} border-red-400 focus:border-red-600 focus:ring-red-600`;

/** Stable per-field id: callers that omit `id` still get a working <label htmlFor>. */
function useFieldId(label: string, explicitId?: string) {
  const autoId = useId();
  return explicitId ?? `field-${label.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "")}-${autoId}`;
}

export function TextField({
  label,
  error,
  ...props
}: InputHTMLAttributes<HTMLInputElement> & { label: string; error?: string }) {
  const id = useFieldId(label, props.id);
  return (
    <div>
      <label htmlFor={id} className="block text-xs font-semibold uppercase tracking-wide text-slate-500">
        {label}
      </label>
      <input id={id} aria-invalid={props["aria-invalid"] ?? (error ? true : undefined)} {...props} className={error ? invalidField : baseField} />
      {error && <p className="mt-1 text-xs font-medium text-red-600">{error}</p>}
    </div>
  );
}

export function SelectField({
  label,
  error,
  children,
  ...props
}: SelectHTMLAttributes<HTMLSelectElement> & { label: string; error?: string; children: ReactNode }) {
  const id = useFieldId(label, props.id);
  return (
    <div>
      <label htmlFor={id} className="block text-xs font-semibold uppercase tracking-wide text-slate-500">
        {label}
      </label>
      <select id={id} aria-invalid={props["aria-invalid"] ?? (error ? true : undefined)} {...props} className={error ? invalidField : baseField}>
        {children}
      </select>
      {error && <p className="mt-1 text-xs font-medium text-red-600">{error}</p>}
    </div>
  );
}

export function TextAreaField({
  label,
  error,
  ...props
}: TextareaHTMLAttributes<HTMLTextAreaElement> & { label: string; error?: string }) {
  const id = useFieldId(label, props.id);
  return (
    <div>
      <label htmlFor={id} className="block text-xs font-semibold uppercase tracking-wide text-slate-500">
        {label}
      </label>
      <textarea id={id} rows={3} aria-invalid={props["aria-invalid"] ?? (error ? true : undefined)} {...props} className={error ? invalidField : baseField} />
      {error && <p className="mt-1 text-xs font-medium text-red-600">{error}</p>}
    </div>
  );
}

/** Standard action row: a form-level error message beside the submit button. */
export function FormActions({
  error,
  success,
  submitting,
  label = "Save",
  onCancel,
}: {
  error?: string | null;
  success?: string | null;
  submitting?: boolean;
  label?: string;
  onCancel?: () => void;
}) {
  return (
    <div className="flex items-center justify-end gap-3 pt-2">
      {error && <p className="mr-auto text-xs font-medium text-red-600">{error}</p>}
      {success && <p className="mr-auto text-xs font-medium text-emerald-600">{success}</p>}
      {onCancel && (
      <button
        type="button"
        onClick={onCancel}
        className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50 min-h-9"
      >
        Cancel
      </button>
      )}
      <button
        type="submit"
        disabled={submitting}
        className="rounded-lg bg-brand-700 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-brand-600 disabled:opacity-50"
      >
        {submitting ? "Saving…" : label}
      </button>
    </div>
  );
}
