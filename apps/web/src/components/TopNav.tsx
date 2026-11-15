"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import {
  AUTH_CHANGED_EVENT,
  createEngineClient,
  loadCachedAuthUser,
  loadEngineSettings,
  saveCachedAuthUser,
} from "@/lib/engine-client";
import type { AuthUser } from "@/lib/types";
import { BrandMark } from "./BrandMark";
import { GitHubMark, OctagonAvatar } from "./GitHubMark";

export function TopNav() {
  const pathname = usePathname();
  const onDeck = pathname?.startsWith("/deck");
  const onPricing = pathname?.startsWith("/pricing");
  const onHome = pathname === "/";

  const [hasToken, setHasToken] = useState(false);
  const [user, setUser] = useState<AuthUser | null>(null);

  useEffect(() => {
    let cancelled = false;

    const refresh = () => {
      const settings = loadEngineSettings();
      const cached = loadCachedAuthUser();
      const signedIn = Boolean(settings.token.trim());
      setHasToken(signedIn);
      setUser(signedIn ? cached : null);

      if (signedIn && !cached) {
        const client = createEngineClient(settings);
        void client
          .getMe()
          .then((me) => {
            if (cancelled) return;
            if (me?.user) {
              saveCachedAuthUser(me.user);
              setUser(me.user);
            }
          })
          .catch(() => {
            // Keep token state untouched if engine is unreachable on landing.
          });
      }
    };

    refresh();
    window.addEventListener(AUTH_CHANGED_EVENT, refresh);
    window.addEventListener("storage", refresh);
    return () => {
      cancelled = true;
      window.removeEventListener(AUTH_CHANGED_EVENT, refresh);
      window.removeEventListener("storage", refresh);
    };
  }, []);

  return (
    <header className="sticky top-0 z-40 border-b border-moss/70 bg-silt/90 backdrop-blur-md">
      <div className="mx-auto flex max-w-7xl items-center justify-between gap-3 px-5 py-3.5 sm:px-8">
        <Link href="/" className="group flex items-center gap-3.5 shrink-0">
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

        <nav
          className="flex flex-wrap items-center justify-end gap-2 sm:gap-3.5"
          aria-label="Primary"
        >
          <Link
            href="/"
            className={`clip-prism-sm px-3.5 py-1.5 text-xs font-semibold uppercase tracking-wider transition-colors ${
              onHome
                ? "bg-basalt text-citron border-b border-citron"
                : "text-ash hover:text-chalk hover:bg-olive"
            }`}
          >
            Overview
          </Link>

          <Link
            href="/pricing"
            className={`clip-prism-sm px-3.5 py-1.5 text-xs font-semibold uppercase tracking-wider transition-colors ${
              onPricing
                ? "bg-basalt text-citron border-b border-citron"
                : "text-ash hover:text-chalk hover:bg-olive"
            }`}
          >
            Pricing
          </Link>

          {hasToken ? (
            <Link
              href="/deck"
              className="clip-octagon-sm bg-cyan/70 p-[1px] transition-transform hover:-translate-y-0.5 active:scale-[0.98]"
              title={user ? `Signed in as @${user.githubLogin}` : "Session active"}
            >
              <span className="flex items-center gap-2 clip-octagon-sm bg-olive px-2.5 py-1 text-xs font-mono text-chalk hover:bg-basalt">
                {user ? (
                  <>
                    <OctagonAvatar
                      login={user.githubLogin}
                      avatarUrl={user.avatarUrl}
                      size={20}
                      tone="cyan"
                    />
                    <span className="max-w-[110px] truncate text-[11px] font-semibold text-cyan">
                      @{user.githubLogin}
                    </span>
                  </>
                ) : (
                  <span className="px-1 text-[11px] font-semibold uppercase tracking-wider text-cyan">
                    Signed in
                  </span>
                )}
              </span>
            </Link>
          ) : !onDeck ? (
            <Link
              href="/deck#signin"
              className="clip-prism-sm bg-moss p-[1px] hover:bg-cyan transition-colors"
            >
              <span className="flex items-center gap-1.5 clip-prism-sm bg-olive px-3.5 py-1.5 text-xs font-semibold uppercase tracking-wider text-chalk hover:bg-basalt hover:text-cyan">
                <GitHubMark size={13} />
                Sign in
              </span>
            </Link>
          ) : null}

          <Link
            href="/deck"
            className={`clip-prism p-[1px] transition-transform active:scale-[0.98] ${
              onDeck ? "bg-citron" : "bg-citron/80 hover:bg-chalk"
            }`}
          >
            <span
              className={`flex items-center gap-2 clip-prism px-4 py-2 text-xs font-bold uppercase tracking-wider sm:px-5 ${
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
