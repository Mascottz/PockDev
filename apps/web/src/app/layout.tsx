import type { Metadata, Viewport } from "next";
import "@fontsource-variable/syne";
import "@fontsource-variable/plus-jakarta-sans";
import "@fontsource/ibm-plex-mono/400.css";
import "@fontsource/ibm-plex-mono/500.css";
import "@fontsource/ibm-plex-mono/600.css";
import "./globals.css";
import { TopNav } from "@/components/TopNav";
import { SiteFooter } from "@/components/SiteFooter";

export const viewport: Viewport = {
  themeColor: "#0C0E0B",
  colorScheme: "dark",
};

export const metadata: Metadata = {
  metadataBase: new URL("https://pockdev.vercel.app"),
  title: {
    default: "PockDev: Catch Every Webhook. Fix Crashes From WhatsApp.",
    template: "%s | PockDev",
  },
  description:
    "Pocket DevOps shield that holds incoming webhooks safely during crashes, links the exact stack line, and opens fix PRs right from WhatsApp.",
  openGraph: {
    type: "website",
    url: "https://pockdev.vercel.app",
    siteName: "PockDev",
    title: "PockDev: Catch Every Webhook. Fix Crashes From WhatsApp.",
    description:
      "Pocket DevOps shield that holds incoming webhooks safely during crashes, links the exact stack line, and opens fix PRs right from WhatsApp.",
    images: [
      {
        url: "/opengraph-image",
        width: 1200,
        height: 630,
        alt: "PockDev: Catch every webhook. Fix crashes on WhatsApp.",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "PockDev: Catch Every Webhook. Fix Crashes From WhatsApp.",
    description:
      "Pocket DevOps shield that holds incoming webhooks safely during crashes, links the exact stack line, and opens fix PRs right from WhatsApp.",
    images: ["/twitter-image"],
  },
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
        <SiteFooter />
      </body>
    </html>
  );
}
