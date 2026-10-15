"use client";

// Register page (Clerk mode only). Rendered when Clerk keys are configured;
// self-registration exists only for guardian/self-enrollment previews — real
// school accounts are still provisioned by admins in the staff directory.
//
// routing="path" requires the Clerk component to own the SUB-routes of its
// mount path too (e.g. /register/verify-email-address for the email-code
// step). A plain page.tsx 404s on those, breaking signup mid-flow — hence
// this catch-all [[...rest]] page rendering <SignUp/> for every depth.

import { SignUp } from "@clerk/nextjs";
import Link from "next/link";
import { isClerkEnabled } from "@/lib/clerk";

export default function RegisterPage() {
  // Same guard as the login page: without keys this deployment is purely
  // session-auth, so self-registration is not offered at all.
  if (!isClerkEnabled()) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-slate-50 px-6 py-12">
        <div className="max-w-sm rounded-xl border border-slate-200 bg-white p-8 text-center shadow-sm">
          <h1 className="text-lg font-bold text-slate-900">Registration unavailable</h1>
          <p className="mt-2 text-sm text-slate-500">
            Accounts are provisioned by the school office. Sign in at{" "}
            <Link href="/login" className="font-semibold text-brand-700 hover:text-brand-600">
              /login
            </Link>
            .
          </p>
        </div>
      </main>
    );
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-50 px-6 py-12">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <Link href="/" className="text-2xl font-extrabold tracking-tight text-brand-900 hover:text-brand-700">
            SchoolHub
          </Link>
          <p className="mt-1 text-sm text-slate-500">Create your account</p>
        </div>
        <div className="flex justify-center">
          {/* No forceRedirectUrl: role-based redirect (by the bridge's
              backend role) applies on sign-up too — parent signups must
              land in the parent portal, not admin. */}
          <SignUp
            path="/register"
            routing="path"
            signInUrl="/login"
            fallbackRedirectUrl="/"
            appearance={{ variables: { colorPrimary: "#2f6f63" } }}
          />
        </div>
      </div>
    </main>
  );
}
