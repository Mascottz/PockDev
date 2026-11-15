import { ImageResponse } from "next/og";
import { SocialCard, loadBrandFonts } from "@/lib/og";

export const runtime = "nodejs";
export const alt = "PockDev: Catch every webhook. Fix crashes on WhatsApp.";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default function TwitterImage() {
  return new ImageResponse(<SocialCard />, { ...size, fonts: loadBrandFonts() });
}
