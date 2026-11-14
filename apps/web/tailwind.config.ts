import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/lib/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        silt: "#0C0E0B",
        olive: "#151913",
        basalt: "#1E241B",
        moss: "#2C3527",
        terracotta: "#E05A33",
        citron: "#C9F23D",
        chalk: "#F3EFE6",
        ash: "#A5A196",
        cyan: "#2AD4C5",
      },
      fontFamily: {
        display: ["var(--font-syne)", "'Syne Variable'", "Syne", "sans-serif"],
        sans: [
          "var(--font-jakarta)",
          "'Plus Jakarta Sans Variable'",
          "Plus Jakarta Sans",
          "sans-serif",
        ],
        mono: ["var(--font-mono)", "'IBM Plex Mono'", "monospace"],
      },
      keyframes: {
        orbitSlow: {
          "0%": { transform: "rotate(0deg)" },
          "100%": { transform: "rotate(360deg)" },
        },
        orbitReverse: {
          "0%": { transform: "rotate(360deg)" },
          "100%": { transform: "rotate(0deg)" },
        },
        breathe: {
          "0%, 100%": { transform: "scale(1)", opacity: "0.75" },
          "50%": { transform: "scale(1.07)", opacity: "1" },
        },
        dashFlow: {
          "0%": { strokeDashoffset: "120" },
          "100%": { strokeDashoffset: "0" },
        },
        trianglePulse: {
          "0%, 100%": { transform: "scale(1)", opacity: "0.9" },
          "50%": { transform: "scale(1.12)", opacity: "1" },
        },
      },
      animation: {
        "orbit-slow": "orbitSlow 28s linear infinite",
        "orbit-med": "orbitSlow 14s linear infinite",
        "orbit-fast": "orbitSlow 7s linear infinite",
        "orbit-reverse": "orbitReverse 19s linear infinite",
        breathe: "breathe 4s ease-in-out infinite",
        "dash-flow": "dashFlow 2.6s linear infinite",
        "triangle-pulse": "trianglePulse 2.2s ease-in-out infinite",
      },
    },
  },
  plugins: [],
};

export default config;
