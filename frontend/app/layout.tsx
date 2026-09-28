import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "SchoolHub",
  description:
    "A portfolio prototype for a school administration and management portal.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-white text-slate-900 antialiased">
        {children}
      </body>
    </html>
  );
}
