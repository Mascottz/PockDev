import type { Metadata } from "next";
import "@fontsource-variable/syne";
import "@fontsource-variable/plus-jakarta-sans";
import "@fontsource/ibm-plex-mono/400.css";
import "@fontsource/ibm-plex-mono/500.css";
import "@fontsource/ibm-plex-mono/600.css";
import "./globals.css";
import { TopNav } from "@/components/TopNav";

export const metadata: Metadata = {
  title: "PockDev: Catch Every Webhook. Fix Crashes From WhatsApp.",
  description:
    "Pocket DevOps shield that holds incoming webhooks safely during crashes, links the exact stack line, and opens fix PRs right from WhatsApp.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-silt text-chalk volcanic-grid selection:bg-citron/30 selection:text-chalk">
        <TopNav />
        <main className="relative z-10">{children}</main>
      </body>
    </html>
  );
}
