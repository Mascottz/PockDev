import { ImageResponse } from "next/og";
import { EmblemSvg, loadBrandFonts } from "@/lib/og";

export const runtime = "nodejs";
export const alt =
  "PockDev: octagon shield with orbital ring, terracotta delta beacon, cyan core";
export const size = { width: 180, height: 180 };
export const contentType = "image/png";

export default function AppleIcon() {
  return new ImageResponse(
    (
      <div
        style={{
          width: 180,
          height: 180,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: "#0C0E0B",
        }}
      >
        <EmblemSvg size={180} />
      </div>
    ),
    { ...size, fonts: loadBrandFonts() },
  );
}
