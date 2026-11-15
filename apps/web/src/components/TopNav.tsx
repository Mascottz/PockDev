"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { BrandMark } from "./BrandMark";

export function TopNav() {
  const pathname = usePathname();
  const onDeck = pathname?.startsWith("/deck");
  const onBrand = pathname?.startsWith("/brand");

  return (
    <header className="sticky top-0 z-40 border-b border-moss/70 bg-silt/90 backdrop-blur-md">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-5 py-3.5 sm:px-8">
        <Link href="/" className="group flex items-center gap-3.5">
          <BrandMark size={38} animated />
          <div>
            <span className="font-display text-lg font-extrabold tracking-tight text-chalk sm:text-xl">
              Pock<span className="text-citron">Dev</span>
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
              !onDeck && !onBrand
                ? "bg-basalt text-citron border-b border-citron"
                : "text-ash hover:text-chalk hover:bg-olive"
            }`}
          >
            Overview
          </Link>

          <Link
            href="/brand"
            className={`clip-prism-sm px-4 py-1.5 text-xs font-semibold uppercase tracking-wider transition-colors ${
              onBrand
                ? "bg-basalt text-citron border-b border-citron"
                : "text-ash hover:text-chalk hover:bg-olive"
            }`}
          >
            Brand Kit
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
