"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { GeometricVessel } from "./GeometricVessel";
import { PrismButton } from "./PrismButton";
import {
  AUTH_CHANGED_EVENT,
  createEngineClient,
  formatNairaFromKobo,
  loadEngineSettings,
} from "@/lib/engine-client";
import type { BillingStatus, EngineSettings } from "@/lib/types";

const FREE_FEATURES = [
  "2 watched apps",
  "5,000 webhook events / month",
  "3-day replay retention",
  "WhatsApp & Email alerts",
  "1/2/3 phone reply loop",
  "@pockdev/sdk crash linking",
];

const PRO_FEATURES = [
  "20 watched apps",
  "100,000 webhook events / month",
  "30-day replay retention",
  "Automated GitHub fix PRs from WhatsApp Reply 1",
  "Multi-workspace isolation",
  "Paystack instant activation",
];

const FAQ_ITEMS = [
  {
    code: "01",
    question: "How fast does Paystack activate Pro Orbit?",
    answer:
      "Immediately after Paystack completes your NGN 15,000 checkout, you return to the Control Deck where PockDev verifies the payment reference and unlocks 20 apps and 100,000 monthly events on the spot.",
  },
  {
    code: "02",
    question: "Do I lose webhooks if my Free Shield workspace hits its cap?",
    answer:
      "No existing watched apps or delivery history are erased. Adding a 3rd app pauses at the plan prompt so you can upgrade in one click and keep shipping.",
  },
  {
    code: "03",
    question: "Can I run multiple client or team workspaces?",
    answer:
      "Yes. Sign in with GitHub to switch or create isolated workspaces from the Control Deck. Billing and usage quotas are tracked cleanly per workspace.",
  },
  {
    code: "04",
    question: "What happens if I cancel or let Pro Orbit expire?",
    answer:
      "Your workspace keeps all captured delivery records and incident PR links intact. Limits step back cleanly to Free Shield until you renew via Paystack.",
  },
];

function FeatureBullet({ accent }: { accent: string }) {
  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 14 14"
      className="mt-1 shrink-0"
      aria-hidden="true"
    >
      <polygon points="7,1 13,7 7,13 1,7" fill={accent} fillOpacity={0.2} stroke={accent} strokeWidth="1.4" />
      <polygon points="7,4.5 9.5,7 7,9.5 4.5,7" fill={accent} />
    </svg>
  );
}

export function PricingView() {
  const router = useRouter();
  const [settings, setSettings] = useState<EngineSettings | null>(null);
  const [billing, setBilling] = useState<BillingStatus | null>(null);
  const [checkingBilling, setCheckingBilling] = useState(false);
  const [upgrading, setUpgrading] = useState(false);
  const [checkoutError, setCheckoutError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;

    const syncAuthAndBilling = () => {
      const loaded = loadEngineSettings();
      if (!active) return;
      setSettings(loaded);

      if (!loaded.token.trim()) {
        setBilling(null);
        setCheckingBilling(false);
        return;
      }

      setCheckingBilling(true);
      const client = createEngineClient(loaded);
      client
        .getBilling()
        .then((status) => {
          if (active) setBilling(status);
        })
        .catch(() => {
          if (active) setBilling(null);
        })
        .finally(() => {
          if (active) setCheckingBilling(false);
        });
    };

    syncAuthAndBilling();
    window.addEventListener(AUTH_CHANGED_EVENT, syncAuthAndBilling);
    window.addEventListener("storage", syncAuthAndBilling);
    return () => {
      active = false;
      window.removeEventListener(AUTH_CHANGED_EVENT, syncAuthAndBilling);
      window.removeEventListener("storage", syncAuthAndBilling);
    };
  }, []);

  const isSignedIn = Boolean(settings?.token.trim());
  const isProActive = billing?.plan === "pro";
  const proPriceLabel = billing
    ? formatNairaFromKobo(billing.proPriceKobo, billing.currency)
    : "NGN 15,000";

  async function handleUpgradeWorkspace() {
    setCheckoutError(null);

    if (!settings || !settings.token.trim()) {
      router.push("/deck#signin");
      return;
    }

    if (isProActive) {
      router.push("/deck");
      return;
    }

    setUpgrading(true);
    try {
      const client = createEngineClient(settings);
      const callbackUrl = `${window.location.origin}/deck?billing=verify`;
      const checkout = await client.startCheckout(callbackUrl);
      if (!checkout.authorizationUrl) {
        throw new Error("Paystack did not return a checkout URL.");
      }
      window.location.assign(checkout.authorizationUrl);
    } catch (err) {
      setCheckoutError(
        err instanceof Error
          ? err.message
          : "Could not start Paystack checkout. Check your engine billing config."
      );
      setUpgrading(false);
    }
  }

  return (
    <div className="mx-auto max-w-7xl px-5 pb-20 pt-10 sm:px-8 sm:pt-14">
      {/* Hero Header */}
      <section className="relative">
        <div className="grid grid-cols-1 items-center gap-8 lg:grid-cols-12">
          <div className="lg:col-span-8">
            <div className="inline-flex items-center gap-2.5 clip-hex-sm bg-olive px-4 py-1.5 font-mono text-[11px] uppercase tracking-[0.22em] text-citron border border-moss">
              <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden="true">
                <polygon points="5,0.5 9.5,5 5,9.5 0.5,5" fill="#C9F23D" />
              </svg>
              Transparent Workspace Billing
            </div>

            <h1 className="mt-5 font-display text-4xl font-extrabold leading-[1.04] tracking-tight text-chalk sm:text-5xl lg:text-6xl">
              Start free. Expand your{" "}
              <span className="text-citron">orbital shield</span> in one click.
            </h1>

            <p className="mt-4 max-w-2xl text-base leading-relaxed text-ash sm:text-lg">
              Every workspace starts on Free Shield with GitHub sign-in, live
              webhook capture, and WhatsApp alerts. Upgrade to Pro Orbit with
              Paystack whenever you need 20 watched apps, 100,000 monthly
              events, and 30-day replay retention.
            </p>
          </div>

          {/* Architectural Geometric Meter */}
          <div className="flex justify-start lg:col-span-4 lg:justify-end">
            <GeometricVessel
              shape="octagon"
              tone="citron"
              className="w-full max-w-xs"
              innerClassName="p-6"
            >
              <div className="flex items-center gap-4">
                <svg
                  width="76"
                  height="76"
                  viewBox="0 0 88 88"
                  className="shrink-0 overflow-visible"
                  aria-hidden="true"
                >
                  <circle
                    cx="44"
                    cy="44"
                    r="38"
                    fill="none"
                    stroke="#363F30"
                    strokeWidth="1.2"
                    strokeDasharray="4 4"
                  />
                  <g className="orbital-cw-slow">
                    <circle
                      cx="44"
                      cy="44"
                      r="29"
                      fill="none"
                      stroke="#C9F23D"
                      strokeWidth="1.6"
                      strokeDasharray="36 18"
                    />
                    <circle cx="44" cy="15" r="3.4" fill="#C9F23D" />
                  </g>
                  <g className="orbital-ccw">
                    <circle
                      cx="44"
                      cy="44"
                      r="20"
                      fill="none"
                      stroke="#2AD4C5"
                      strokeWidth="1.4"
                      strokeDasharray="18 14"
                    />
                  </g>
                  <polygon
                    points="44,31 55,44 44,57 33,44"
                    fill="#141813"
                    stroke="#C9F23D"
                    strokeWidth="1.8"
                  />
                  <polygon points="44,38 49,44 44,50 39,44" fill="#C9F23D" />
                </svg>

                <div>
                  <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-ash">
                    Paystack NGN
                  </p>
                  <p className="mt-1 font-display text-2xl font-extrabold text-chalk">
                    {proPriceLabel}
                  </p>
                  <p className="mt-0.5 font-mono text-[11px] uppercase tracking-[0.16em] text-citron">
                    Per workspace / mo
                  </p>
                </div>
              </div>

              <div className="mt-4 border-t border-moss/70 pt-3 font-mono text-[10px] uppercase tracking-[0.18em] text-ash">
                {checkingBilling
                  ? "Checking workspace status..."
                  : isProActive
                    ? "Current workspace: Pro Orbit Active"
                    : isSignedIn
                      ? "Signed in: 1-click Paystack checkout ready"
                      : "Sign in on Control Deck to upgrade"}
              </div>
            </GeometricVessel>
          </div>
        </div>
      </section>

      {checkoutError ? (
        <div className="mt-8">
          <GeometricVessel
            shape="hex-bar"
            tone="terracotta"
            innerClassName="px-6 py-3.5"
          >
            <div className="flex flex-wrap items-center justify-between gap-3">
              <p className="font-mono text-xs text-terracotta">
                {checkoutError}
              </p>
              <button
                type="button"
                onClick={() => setCheckoutError(null)}
                className="clip-prism-sm bg-basalt px-3 py-1 font-mono text-[10px] uppercase tracking-wider text-chalk hover:bg-moss"
              >
                Dismiss
              </button>
            </div>
          </GeometricVessel>
        </div>
      ) : null}

      {/* Two Geometric Tier Polygons */}
      <section className="mt-10 grid grid-cols-1 gap-7 lg:grid-cols-2">
        {/* FREE SHIELD */}
        <GeometricVessel
          shape="diamond-cut"
          tone="moss"
          className="h-full"
          innerClassName="flex h-full flex-col justify-between p-7 sm:p-9"
        >
          <div>
            <div className="flex items-start justify-between gap-4">
              <div>
                <span className="clip-hex-sm inline-block bg-basalt px-3 py-1 font-mono text-[10px] font-bold uppercase tracking-[0.22em] text-cyan">
                  TIER 01 : FREE SHIELD
                </span>
                <h2 className="mt-3 font-display text-3xl font-extrabold text-chalk">
                  Free Shield
                </h2>
                <p className="mt-1.5 text-sm text-ash">
                  Built for side projects, staging webhooks, and solo launches.
                </p>
              </div>

              <svg
                width="52"
                height="52"
                viewBox="0 0 56 56"
                className="shrink-0"
                aria-hidden="true"
              >
                <polygon
                  points="28,4 48,14 48,34 28,52 8,34 8,14"
                  fill="#141813"
                  stroke="#2AD4C5"
                  strokeWidth="1.8"
                />
                <polygon
                  points="28,17 38,34 18,34"
                  fill="none"
                  stroke="#2AD4C5"
                  strokeWidth="1.8"
                />
              </svg>
            </div>

            <div className="mt-7 flex items-baseline gap-2.5 border-y border-moss/70 py-5">
              <span className="font-display text-4xl font-extrabold tracking-tight text-chalk sm:text-5xl">
                NGN 0
              </span>
              <span className="font-mono text-xs uppercase tracking-[0.2em] text-ash">
                / forever
              </span>
            </div>

            {/* Capacity Strip */}
            <div className="mt-6 grid grid-cols-3 gap-2.5">
              <div className="clip-octagon-sm bg-basalt px-3 py-3 text-center">
                <p className="font-display text-lg font-extrabold text-chalk">
                  2
                </p>
                <p className="font-mono text-[10px] uppercase tracking-wider text-ash">
                  Apps
                </p>
              </div>
              <div className="clip-octagon-sm bg-basalt px-3 py-3 text-center">
                <p className="font-display text-lg font-extrabold text-chalk">
                  5,000
                </p>
                <p className="font-mono text-[10px] uppercase tracking-wider text-ash">
                  Events/mo
                </p>
              </div>
              <div className="clip-octagon-sm bg-basalt px-3 py-3 text-center">
                <p className="font-display text-lg font-extrabold text-chalk">
                  3 days
                </p>
                <p className="font-mono text-[10px] uppercase tracking-wider text-ash">
                  Retention
                </p>
              </div>
            </div>

            <ul className="mt-7 space-y-3">
              {FREE_FEATURES.map((feature) => (
                <li
                  key={feature}
                  className="flex items-start gap-3 text-sm text-chalk"
                >
                  <FeatureBullet accent="#2AD4C5" />
                  <span>{feature}</span>
                </li>
              ))}
            </ul>
          </div>

          <div className="mt-9 pt-6 border-t border-moss/60">
            <Link href="/deck" className="block">
              <PrismButton
                type="button"
                variant="olive"
                size="lg"
                className="w-full"
              >
                {isSignedIn ? "Open Control Deck" : "Start Free on GitHub"}
              </PrismButton>
            </Link>
            <p className="mt-3 text-center font-mono text-[10px] uppercase tracking-[0.18em] text-ash">
              No card required. Activate in under 60 seconds.
            </p>
          </div>
        </GeometricVessel>

        {/* PRO ORBIT */}
        <GeometricVessel
          shape="octagon"
          tone="citron"
          className="h-full"
          innerClassName="flex h-full flex-col justify-between bg-olive p-7 sm:p-9"
        >
          <div>
            <div className="flex items-start justify-between gap-4">
              <div>
                <div className="flex flex-wrap items-center gap-2">
                  <span className="clip-hex-sm inline-block bg-citron px-3 py-1 font-mono text-[10px] font-bold uppercase tracking-[0.22em] text-silt">
                    TIER 02 : PRO ORBIT
                  </span>
                  {isProActive ? (
                    <span className="clip-hex-sm inline-block bg-cyan/20 px-2.5 py-1 font-mono text-[10px] font-bold uppercase tracking-[0.18em] text-cyan">
                      Active Plan
                    </span>
                  ) : null}
                </div>
                <h2 className="mt-3 font-display text-3xl font-extrabold text-chalk">
                  Pro Orbit
                </h2>
                <p className="mt-1.5 text-sm text-ash">
                  Full-scale webhook armor and automated WhatsApp fix PRs for
                  production teams.
                </p>
              </div>

              <svg
                width="54"
                height="54"
                viewBox="0 0 56 56"
                className="shrink-0 overflow-visible"
                aria-hidden="true"
              >
                <circle
                  cx="28"
                  cy="28"
                  r="23"
                  fill="#141813"
                  stroke="#C9F23D"
                  strokeWidth="1.6"
                  strokeDasharray="16 6"
                />
                <polygon
                  points="28,11 41,28 28,45 15,28"
                  fill="#C9F23D"
                  fillOpacity={0.18}
                  stroke="#C9F23D"
                  strokeWidth="1.8"
                />
                <circle cx="28" cy="28" r="4.5" fill="#C9F23D" />
              </svg>
            </div>

            <div className="mt-7 flex items-baseline gap-2.5 border-y border-citron/35 py-5">
              <span className="font-display text-4xl font-extrabold tracking-tight text-citron sm:text-5xl">
                {proPriceLabel}
              </span>
              <span className="font-mono text-xs uppercase tracking-[0.2em] text-ash">
                / month
              </span>
            </div>

            {/* Capacity Strip */}
            <div className="mt-6 grid grid-cols-3 gap-2.5">
              <div className="clip-octagon-sm bg-basalt px-3 py-3 text-center border border-citron/25">
                <p className="font-display text-lg font-extrabold text-citron">
                  20
                </p>
                <p className="font-mono text-[10px] uppercase tracking-wider text-ash">
                  Apps
                </p>
              </div>
              <div className="clip-octagon-sm bg-basalt px-3 py-3 text-center border border-citron/25">
                <p className="font-display text-lg font-extrabold text-citron">
                  100,000
                </p>
                <p className="font-mono text-[10px] uppercase tracking-wider text-ash">
                  Events/mo
                </p>
              </div>
              <div className="clip-octagon-sm bg-basalt px-3 py-3 text-center border border-citron/25">
                <p className="font-display text-lg font-extrabold text-citron">
                  30 days
                </p>
                <p className="font-mono text-[10px] uppercase tracking-wider text-ash">
                  Retention
                </p>
              </div>
            </div>

            <ul className="mt-7 space-y-3">
              {PRO_FEATURES.map((feature) => (
                <li
                  key={feature}
                  className="flex items-start gap-3 text-sm text-chalk"
                >
                  <FeatureBullet accent="#C9F23D" />
                  <span>{feature}</span>
                </li>
              ))}
            </ul>
          </div>

          <div className="mt-9 pt-6 border-t border-citron/30">
            <PrismButton
              type="button"
              variant="citron"
              size="lg"
              className="w-full"
              disabled={upgrading}
              onClick={handleUpgradeWorkspace}
            >
              {upgrading
                ? "Redirecting to Paystack..."
                : isProActive
                  ? "Manage Pro Workspace in Deck"
                  : "Upgrade Workspace"}
            </PrismButton>
            <p className="mt-3 text-center font-mono text-[10px] uppercase tracking-[0.18em] text-ash">
              {isSignedIn
                ? "Instant Paystack checkout for your active workspace."
                : "Sign in with GitHub on Control Deck first, then upgrade in one click."}
            </p>
          </div>
        </GeometricVessel>
      </section>

      {/* Guarantee Strip */}
      <section className="mt-12">
        <GeometricVessel
          shape="hex-bar"
          tone="cyan"
          innerClassName="px-6 py-5 sm:px-9"
        >
          <div className="grid grid-cols-1 gap-5 md:grid-cols-3 md:items-center">
            <div className="flex items-center gap-3.5">
              <svg width="26" height="26" viewBox="0 0 26 26" aria-hidden="true">
                <polygon
                  points="13,2 24,8 24,18 13,24 2,18 2,8"
                  fill="none"
                  stroke="#2AD4C5"
                  strokeWidth="1.6"
                />
                <polyline
                  points="8,13 11.5,16.5 18,10"
                  fill="none"
                  stroke="#C9F23D"
                  strokeWidth="2"
                />
              </svg>
              <div>
                <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan">
                  Zero Lock-In
                </p>
                <p className="text-sm font-semibold text-chalk">
                  Paystack NGN billing, cancel anytime
                </p>
              </div>
            </div>

            <div className="flex items-center gap-3.5">
              <svg width="26" height="26" viewBox="0 0 26 26" aria-hidden="true">
                <circle
                  cx="13"
                  cy="13"
                  r="10"
                  fill="none"
                  stroke="#C9F23D"
                  strokeWidth="1.6"
                  strokeDasharray="6 3"
                />
                <polygon points="13,7 17,13 13,19 9,13" fill="#C9F23D" />
              </svg>
              <div>
                <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-citron">
                  Instant Unlock
                </p>
                <p className="text-sm font-semibold text-chalk">
                  Automatic verification on return to Deck
                </p>
              </div>
            </div>

            <div className="flex items-center gap-3.5">
              <svg width="26" height="26" viewBox="0 0 26 26" aria-hidden="true">
                <polygon
                  points="13,3 23,21 3,21"
                  fill="none"
                  stroke="#E05A33"
                  strokeWidth="1.6"
                />
                <circle cx="13" cy="15" r="2.2" fill="#E05A33" />
              </svg>
              <div>
                <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-terracotta">
                  Zero Dropped Events
                </p>
                <p className="text-sm font-semibold text-chalk">
                  Full payload preservation across upgrades
                </p>
              </div>
            </div>
          </div>
        </GeometricVessel>
      </section>

      {/* Geometric FAQ Section */}
      <section className="mt-14">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div>
            <p className="font-mono text-[11px] uppercase tracking-[0.24em] text-citron">
              Billing Questions
            </p>
            <h2 className="mt-2 font-display text-2xl font-extrabold text-chalk sm:text-3xl">
              Clear answers before you upgrade
            </h2>
          </div>

          <Link
            href="/deck"
            className="clip-prism-sm bg-basalt px-4 py-2 font-mono text-[11px] uppercase tracking-[0.18em] text-ash transition-colors hover:bg-citron hover:text-silt"
          >
            Go to Control Deck -&gt;
          </Link>
        </div>

        <div className="mt-6 grid grid-cols-1 gap-5 md:grid-cols-2">
          {FAQ_ITEMS.map((item) => (
            <GeometricVessel
              key={item.code}
              shape="octagon-sm"
              tone="moss"
              innerClassName="p-6"
            >
              <div className="flex items-start gap-3.5">
                <span className="clip-hex-sm shrink-0 bg-basalt px-2.5 py-1 font-mono text-[11px] font-bold text-citron">
                  {item.code}
                </span>
                <div>
                  <h3 className="font-display text-base font-bold text-chalk">
                    {item.question}
                  </h3>
                  <p className="mt-2 text-sm leading-relaxed text-ash">
                    {item.answer}
                  </p>
                </div>
              </div>
            </GeometricVessel>
          ))}
        </div>
      </section>
    </div>
  );
}
