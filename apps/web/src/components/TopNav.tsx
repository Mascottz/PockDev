"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

export function TopNav() {
  const pathname = usePathname();
  const onDeck = pathname?.startsWith("/deck");

  return (
    <header className="sticky top-0 z-40 border-b border-moss/70 bg-silt/90 backdrop-blur-md">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-5 py-3.5 sm:px-8">
        <Link href="/" className="group flex items-center gap-3.5">
          <svg
            width="38"
            height="38"
            viewBox="0 0 48 48"
            className="overflow-visible"
            aria-hidden="true"
          >
            <g className="orbital-cw-slow">
              <circle
                cx="24"
                cy="24"
                r="21"
                fill="none"
                stroke="#C9F23D"
                strokeWidth="1.3"
                strokeDasharray="16 8 4 8"
              />
              <circle cx="24" cy="3" r="2.6" fill="#C9F23D" />
            </g>
            <circle
              cx="24"
              cy="24"
              r="14"
              fill="#151913"
              stroke="#2C3527"
              strokeWidth="1.2"
            />
            <polygon
              points="24,14 33,30 15,30"
              fill="none"
              stroke="#E05A33"
              strokeWidth="2"
              className="transition-transform duration-300 group-hover:scale-110 svg-origin-center"
            />
            <circle cx="24" cy="25" r="2.4" fill="#2AD4C5" />
          </svg>
          <div>
            <span className="font-display text-lg font-extrabold tracking-tight text-chalk sm:text-xl">
              PockDev
            </span>
            <span className="block font-mono text-[10px] uppercase tracking-[0.22em] text-ash">
              Pocket Reliability Shield
            </span>
          </div>
        </Link>

        <nav className="flex items-center gap-2.5 sm:gap-4" aria-label="Primary">
          <Link
            href="/"
            className={`clip-prism-sm px-4 py-1.5 text-xs font-semibold uppercase tracking-wider transition-colors ${
              !onDeck
                ? "bg-basalt text-citron border-b border-citron"
                : "text-ash hover:text-chalk hover:bg-olive"
            }`}
          >
            Overview
          </Link>

          <Link
            href="/deck"
            className={`clip-prism p-[1px] transition-transform active:scale-[0.98] ${
              onDeck ? "bg-citron" : "bg-citron/80 hover:bg-chalk"
            }`}
          >
            <span
              className={`flex items-center gap-2 clip-prism px-5 py-2 text-xs font-bold uppercase tracking-wider ${
                onDeck
                  ? "bg-citron text-silt"
                  : "bg-olive text-citron hover:bg-citron hover:text-silt"
              }`}
            >
              <svg width="10" height="10" viewBox="0 0 12 12" aria-hidden="true">
                <polygon points="6,1 11,10 1,10" fill="currentColor" />
              </svg>
              Control Deck
            </span>
          </Link>
        </nav>
      </div>
    </header>
  );
}
