import type { Metadata } from "next";
import "./globals.css";
import { AuthProvider } from "@/components/auth-provider";
import { ClerkBridge } from "@/components/clerk-bridge";

export const metadata: Metadata = {
  title: "SchoolHub Academy",
  description:
    "SchoolHub Academy — an independent day school where bright futures begin. Admissions, academics, news, and the parent/student/staff portal.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-white text-slate-900 antialiased">
        <AuthProvider>
          <ClerkBridge>{children}</ClerkBridge>
        </AuthProvider>
      </body>
    </html>
  );
}
