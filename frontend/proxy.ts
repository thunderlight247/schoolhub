// Next.js edge middleware (proxy). Only active when Clerk keys are configured;
// with no keys present it passes every request through untouched so the plain
// session-auth deployment behaves exactly as before.
//
// The '/__clerk/:path*' matcher is Clerk's auto-proxy path required by the
// Next.js integration (see the Clerk Next.js quickstart).
//
// NOTE: The __clerk matcher is always registered (required by Clerk's
// integration); classic-mode dev sessions must finish browser checks inside
// the dev-server survival window (see remember.md gotchas).

import { clerkMiddleware, createRouteMatcher } from "@clerk/nextjs/server";
import { NextResponse, type NextMiddleware } from "next/server";

const clerkEnabled = !!process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY;
const isPublicRoute = createRouteMatcher(["/", "/login", "/register"]);

const clerkAuthMiddleware = clerkMiddleware(async (auth, req) => {
  if (!isPublicRoute(req)) {
    await auth.protect();
  }
});

export default function middleware(
  req: Parameters<NextMiddleware>[0],
  event: Parameters<NextMiddleware>[1],
): ReturnType<NextMiddleware> {
  if (!clerkEnabled) {
    return NextResponse.next();
  }
  return clerkAuthMiddleware(req, event);
}

export const config = {
  matcher: ["/((?!_next|[^?]*\\..*).*)", "/(api|trpc)(.*)", "/__clerk/:path*"],
};
