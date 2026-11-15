import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "PockDev: Webhook Shield & Pocket SRE",
    short_name: "PockDev",
    description:
      "Pocket DevOps shield that holds incoming webhooks safely during crashes, links the exact stack line, and opens fix PRs right from WhatsApp.",
    start_url: "/deck",
    display: "standalone",
    background_color: "#0C0E0B",
    theme_color: "#0C0E0B",
    icons: [
      { src: "/icon.svg", sizes: "any", type: "image/svg+xml" },
      { src: "/apple-icon", sizes: "180x180", type: "image/png" },
    ],
  };
}
