"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { GeometricVessel } from "./GeometricVessel";
import { PrismButton } from "./PrismButton";

interface SimStep {
  id: number;
  code: string;
  label: string;
  headline: string;
  body: string;
  statusTag: string;
  tone: "citron" | "terracotta" | "cyan";
}

const STEPS: SimStep[] = [
  {
    id: 0,
    code: "01",
    label: "Webhook Caught",
    headline: "Shield catches every payment event first",
    body: "Paystack fires charge.success to your PockDev Shield URL. PockDev stores the exact body on disk and object storage, stamps X-PockDev-Delivery-Id, and forwards it to your app.",
    statusTag: "Shield active: 0 events lost",
    tone: "citron",
  },
  {
    id: 1,
    code: "02",
    label: "App Crash Linked",
    headline: "Crash stack links to the paused webhook",
    body: "Your handler throws at src/webhooks/paystack.ts:47 because data.amount moved to data.charged_amount. @pockdev/sdk captures the stack and delivery ID, and Shield safely holds 6 incoming events.",
    statusTag: "6 events safely held",
    tone: "terracotta",
  },
  {
    id: 2,
    code: "03",
    label: "Plain WhatsApp Diagnosis",
    headline: "Calm plain-English alert lands on WhatsApp",
    body: "Instead of a noisy 40-line stack trace, PockDev reads the changed payload and recent commit 8a4f2c1, then sends one short message you can act on from anywhere.",
    statusTag: "WhatsApp waiting: Reply 1, 2, or 3",
    tone: "cyan",
  },
  {
    id: 3,
    code: "04",
    label: "Reply 1: Fix PR + Replay",
    headline: "One digit opens the PR and replays held events",
    body: "You reply 1 right inside WhatsApp. PockDev drafts PR #48 updating line 47 to read data.charged_amount ?? data.amount, then sends all 6 held webhooks cleanly to your app.",
    statusTag: "All quiet: 6 held events delivered",
    tone: "citron",
  },
];

type SnippetFramework = "nextjs" | "express" | "hono";

const SNIPPETS: Record<SnippetFramework, string> = {
  nextjs: `import { init, wrap } from "@pockdev/sdk";

init({ key: process.env.POCKDEV_KEY!, endpoint: "https://engine.pockdev.app" });

export const POST = wrap(async (req: Request) => {
  const event = await req.json();
  await settleCharge(event.data);
  return Response.json({ ok: true });
});`,
  express: `import { init, wrap } from "@pockdev/sdk";

init({ key: process.env.POCKDEV_KEY!, endpoint: "https://engine.pockdev.app" });

app.post("/webhooks/paystack", wrap(async (req, res) => {
  await settleCharge(req.body.data);
  res.json({ ok: true });
}));`,
  hono: `import { init, wrap } from "@pockdev/sdk";

init({ key: process.env.POCKDEV_KEY!, endpoint: "https://engine.pockdev.app" });

app.post("/webhooks/paystack", wrap(async (c) => {
  const event = await c.req.json();
  await settleCharge(event.data);
  return c.json({ ok: true });
}));`,
};

export function LandingHero() {
  const [stepIndex, setStepIndex] = useState(0);
  const [autoPlay, setAutoPlay] = useState(true);
  const [framework, setFramework] = useState<SnippetFramework>("nextjs");
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!autoPlay) return;
    const timer = window.setInterval(() => {
      setStepIndex((prev) => (prev + 1) % STEPS.length);
    }, 5200);
    return () => window.clearInterval(timer);
  }, [autoPlay]);

  const active = STEPS[stepIndex];
  const isCrashing = stepIndex === 1 || stepIndex === 2;
  const isFixing = stepIndex === 3;

  const selectStep = (idx: number) => {
    setAutoPlay(false);
    setStepIndex(idx);
  };

  const handleCopySnippet = async () => {
    try {
      await navigator.clipboard.writeText(SNIPPETS[framework]);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      // Clipboard blocked in restricted browser contexts.
    }
  };

  return (
    <div className="mx-auto max-w-7xl px-5 pb-24 pt-10 sm:px-8 sm:pt-14">
      {/* Top asymmetric geometric headline + action bar */}
      <section className="grid gap-10 lg:grid-cols-12 lg:items-end">
        <div className="lg:col-span-8">
          <div className="mb-5 inline-flex items-center gap-3">
            <span className="clip-hex-sm bg-terracotta/90 p-[1px]">
              <span className="flex items-center gap-2 clip-hex-sm bg-olive px-3.5 py-1 font-mono text-[11px] uppercase tracking-[0.18em] text-terracotta">
                <svg width="8" height="8" viewBox="0 0 10 10" aria-hidden="true">
                  <polygon points="5,1 9,9 1,9" fill="#E05A33" />
                </svg>
                Zero-lost webhooks
              </span>
            </span>
            <span className="font-mono text-xs uppercase tracking-[0.16em] text-ash">
              Partner -&gt; Shield -&gt; App -&gt; WhatsApp
            </span>
          </div>

          <h1 className="font-display text-4xl font-extrabold leading-[1.03] tracking-tight text-chalk sm:text-6xl lg:text-[64px]">
            Catch every webhook.
            <span className="mt-1 block text-citron">
              Fix crashes from WhatsApp.
            </span>
          </h1>

          <p className="mt-5 max-w-2xl text-base leading-relaxed text-ash sm:text-lg">
            PockDev stands in front of your webhook endpoints, holds incoming
            events safely when your code throws, links the exact stack line, and
            drafts a fix PR when you reply{" "}
            <span className="font-mono font-semibold text-chalk">1</span> on
            WhatsApp.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-3.5 lg:col-span-4 lg:justify-end">
          <Link href="/deck">
            <PrismButton variant="citron" size="lg">
              Open Control Deck
              <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
                <polygon points="2,1 11,6 2,11" fill="currentColor" />
              </svg>
            </PrismButton>
          </Link>
          <PrismButton
            variant="olive"
            size="lg"
            onClick={() => setAutoPlay((v) => !v)}
          >
            {autoPlay ? "Pause drill" : "Resume drill"}
          </PrismButton>
        </div>
      </section>

      {/* Centerpiece: Animated 4-step Orbital + Triangular SVG Flow Simulation */}
      <section className="mt-12">
        <GeometricVessel
          shape="octagon"
          tone={active.tone}
          innerClassName="p-5 sm:p-8"
        >
          {/* Step trigger prisms */}
          <div className="grid grid-cols-2 gap-2.5 lg:grid-cols-4">
            {STEPS.map((step, idx) => {
              const selected = idx === stepIndex;
              const toneBorder =
                step.tone === "terracotta"
                  ? "bg-terracotta"
                  : step.tone === "cyan"
                    ? "bg-cyan"
                    : "bg-citron";
              return (
                <button
                  key={step.id}
                  type="button"
                  onClick={() => selectStep(idx)}
                  className={`clip-prism-sm p-[1px] text-left transition-transform active:scale-[0.99] ${
                    selected ? toneBorder : "bg-moss hover:bg-ash/50"
                  }`}
                >
                  <div
                    className={`clip-prism-sm px-4 py-3 ${
                      selected ? "bg-basalt" : "bg-silt/85 hover:bg-olive"
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-mono text-[11px] font-semibold tracking-widest text-ash">
                        STEP {step.code}
                      </span>
                      {selected ? (
                        <svg width="10" height="10" viewBox="0 0 12 12" aria-hidden="true">
                          <polygon
                            points="6,1 11,10 1,10"
                            fill={
                              step.tone === "terracotta"
                                ? "#E05A33"
                                : step.tone === "cyan"
                                  ? "#2AD4C5"
                                  : "#C9F23D"
                            }
                          />
                        </svg>
                      ) : null}
                    </div>
                    <p
                      className={`mt-1 font-display text-sm font-bold ${
                        selected ? "text-chalk" : "text-ash"
                      }`}
                    >
                      {step.label}
                    </p>
                  </div>
                </button>
              );
            })}
          </div>

          {/* Live animated SVG flow diagram */}
          <div className="mt-7 overflow-x-auto border-y border-moss/60 py-4">
            <svg
              viewBox="0 0 980 280"
              className="mx-auto h-auto w-full min-w-[680px] max-w-[980px]"
              role="img"
              aria-label="Live PockDev flow from Partner to PockDev Shield to Your App to WhatsApp"
            >
              {/* Faint background orbital calibration arcs */}
              <circle
                cx="340"
                cy="140"
                r="116"
                fill="none"
                stroke="#2C3527"
                strokeWidth="1"
                strokeDasharray="3 9"
              />
              <circle
                cx="645"
                cy="140"
                r="92"
                fill="none"
                stroke="#2C3527"
                strokeWidth="1"
                strokeDasharray="3 9"
              />

              {/* Conduit 1: Partner -> PockDev Shield */}
              <path
                d="M 155 140 L 252 140"
                fill="none"
                stroke="#2C3527"
                strokeWidth="2.5"
              />
              <path
                d="M 155 140 L 252 140"
                fill="none"
                stroke="#C9F23D"
                strokeWidth="3"
                className={
                  stepIndex === 0 ? "flow-conduit-active" : "flow-conduit-calm"
                }
              />

              {/* Conduit 2: PockDev Shield -> Your App */}
              <path
                d="M 426 140 L 570 140"
                fill="none"
                stroke="#2C3527"
                strokeWidth="2.5"
              />
              <path
                d="M 426 140 L 570 140"
                fill="none"
                stroke={
                  isCrashing ? "#E05A33" : isFixing ? "#C9F23D" : "#C9F23D"
                }
                strokeWidth="3"
                className={
                  isCrashing
                    ? "flow-conduit-alert"
                    : isFixing
                      ? "flow-conduit-active"
                      : "flow-conduit-calm"
                }
              />

              {/* Conduit 3: Your App -> WhatsApp (Diagnosis path) */}
              <path
                d="M 696 112 L 758 84 L 818 112"
                fill="none"
                stroke="#2C3527"
                strokeWidth="2.5"
              />
              <path
                d="M 696 112 L 758 84 L 818 112"
                fill="none"
                stroke={stepIndex >= 2 ? "#2AD4C5" : "#2C3527"}
                strokeWidth="3"
                className={stepIndex === 2 ? "flow-conduit-active" : "flow-conduit-calm"}
              />

              {/* Conduit 4: WhatsApp -> PockDev Shield (Reply 1 Fix + Replay return path) */}
              <path
                d="M 858 196 L 858 244 L 340 244 L 340 218"
                fill="none"
                stroke="#2C3527"
                strokeWidth="2"
                strokeDasharray="4 6"
              />
              {isFixing ? (
                <path
                  d="M 858 196 L 858 244 L 340 244 L 340 218"
                  fill="none"
                  stroke="#2AD4C5"
                  strokeWidth="3.2"
                  className="flow-conduit-active"
                />
              ) : null}

              {/* NODE 1: Partner (Hexagon) */}
              <g transform="translate(88, 140)">
                <polygon
                  points="-56,0 -34,-44 34,-44 56,0 34,44 -34,44"
                  fill="#151913"
                  stroke={stepIndex === 0 ? "#C9F23D" : "#2C3527"}
                  strokeWidth={stepIndex === 0 ? 2.4 : 1.5}
                />
                <polygon
                  points="-42,0 -25,-32 25,-32 42,0 25,32 -25,32"
                  fill="none"
                  stroke="#2C3527"
                  strokeWidth="1"
                />
                <text
                  x="0"
                  y="-4"
                  textAnchor="middle"
                  fill="#F3EFE6"
                  fontSize="13"
                  fontWeight="700"
                >
                  Partner
                </text>
                <text
                  x="0"
                  y="14"
                  textAnchor="middle"
                  fill="#A5A196"
                  fontSize="10"
                  fontFamily="monospace"
                >
                  charge.success
                </text>
              </g>

              {/* NODE 2: PockDev Shield (Breathing concentric orbital rings) */}
              <g transform="translate(340, 140)">
                <circle
                  cx="0"
                  cy="0"
                  r="80"
                  fill="none"
                  stroke={isCrashing ? "#E05A33" : "#C9F23D"}
                  strokeOpacity={0.35}
                  strokeWidth="1.2"
                  strokeDasharray="6 8"
                  className="orbital-breath"
                />
                <g className={isCrashing ? "orbital-cw-fast" : "orbital-cw-slow"}>
                  <circle
                    cx="0"
                    cy="0"
                    r="64"
                    fill="none"
                    stroke={isCrashing ? "#E05A33" : "#C9F23D"}
                    strokeWidth="1.8"
                    strokeDasharray="44 18 12 18"
                  />
                  {/* Held event satellites */}
                  <circle cx="0" cy="-64" r="4.5" fill={isCrashing ? "#E05A33" : "#C9F23D"} />
                  <circle cx="56" cy="30" r="3.5" fill="#2AD4C5" />
                  {isCrashing ? (
                    <>
                      <circle cx="-56" cy="30" r="4" fill="#E05A33" />
                      <circle cx="-44" cy="-46" r="3.5" fill="#E05A33" />
                      <circle cx="44" cy="-46" r="3.5" fill="#E05A33" />
                      <circle cx="0" cy="64" r="3.5" fill="#E05A33" />
                    </>
                  ) : null}
                </g>
                <g className="orbital-ccw">
                  <circle
                    cx="0"
                    cy="0"
                    r="48"
                    fill="#151913"
                    stroke={isFixing ? "#2AD4C5" : isCrashing ? "#E05A33" : "#C9F23D"}
                    strokeWidth="2"
                  />
                </g>
                <text
                  x="0"
                  y="-6"
                  textAnchor="middle"
                  fill="#F3EFE6"
                  fontSize="13"
                  fontWeight="800"
                >
                  PockDev
                </text>
                <text
                  x="0"
                  y="11"
                  textAnchor="middle"
                  fill={isCrashing ? "#E05A33" : isFixing ? "#2AD4C5" : "#C9F23D"}
                  fontSize="10"
                  fontFamily="monospace"
                >
                  {isCrashing ? "6 HELD SAFE" : isFixing ? "REPLAY 200" : "SHIELD LIVE"}
                </text>
              </g>

              {/* NODE 3: Your App (Transforms into pulsing Terracotta Triangle when crashing!) */}
              <g transform="translate(645, 140)">
                {isCrashing ? (
                  <>
                    <polygon
                      points="0,-64 58,42 -58,42"
                      fill="none"
                      stroke="#E05A33"
                      strokeWidth="1.5"
                      className="beacon-shockwave"
                    />
                    <polygon
                      points="0,-58 52,38 -52,38"
                      fill="#2A140E"
                      stroke="#E05A33"
                      strokeWidth="2.5"
                    />
                    <text
                      x="0"
                      y="2"
                      textAnchor="middle"
                      fill="#F3EFE6"
                      fontSize="12"
                      fontWeight="700"
                    >
                      Your App
                    </text>
                    <text
                      x="0"
                      y="19"
                      textAnchor="middle"
                      fill="#E05A33"
                      fontSize="10"
                      fontFamily="monospace"
                    >
                      paystack.ts:47
                    </text>
                  </>
                ) : (
                  <>
                    <circle
                      cx="0"
                      cy="0"
                      r="52"
                      fill="#151913"
                      stroke={isFixing ? "#C9F23D" : "#2C3527"}
                      strokeWidth={isFixing ? 2.4 : 1.6}
                    />
                    <g className="orbital-cw-med">
                      <circle
                        cx="0"
                        cy="0"
                        r="42"
                        fill="none"
                        stroke={isFixing ? "#C9F23D" : "#2AD4C5"}
                        strokeOpacity={0.55}
                        strokeDasharray="18 12"
                      />
                    </g>
                    <text
                      x="0"
                      y="-4"
                      textAnchor="middle"
                      fill="#F3EFE6"
                      fontSize="13"
                      fontWeight="700"
                    >
                      Your App
                    </text>
                    <text
                      x="0"
                      y="14"
                      textAnchor="middle"
                      fill="#C9F23D"
                      fontSize="10"
                      fontFamily="monospace"
                    >
                      {isFixing ? "PR #48 FIXED" : "HTTP 200 OK"}
                    </text>
                  </>
                )}
              </g>

              {/* NODE 4: WhatsApp Handset (Chamfered Octagon) */}
              <g transform="translate(868, 140)">
                <polygon
                  points="-42,-54 42,-54 58,-38 58,38 42,54 -42,54 -58,38 -58,-38"
                  fill="#151913"
                  stroke={stepIndex >= 2 ? "#2AD4C5" : "#2C3527"}
                  strokeWidth={stepIndex >= 2 ? 2.4 : 1.5}
                />
                <polygon
                  points="-34,-44 34,-44 46,-32 46,32 34,44 -34,44 -46,32 -46,-32"
                  fill="none"
                  stroke="#2C3527"
                  strokeWidth="1"
                />
                <text
                  x="0"
                  y="-6"
                  textAnchor="middle"
                  fill="#F3EFE6"
                  fontSize="13"
                  fontWeight="700"
                >
                  WhatsApp
                </text>
                <text
                  x="0"
                  y="13"
                  textAnchor="middle"
                  fill={stepIndex >= 2 ? "#2AD4C5" : "#A5A196"}
                  fontSize="10"
                  fontFamily="monospace"
                >
                  {stepIndex === 3
                    ? "REPLIED: 1"
                    : stepIndex === 2
                      ? "REPLY 1 / 2 / 3"
                      : "STANDBY"}
                </text>
              </g>

              {/* Return conduit caption */}
              <text
                x="600"
                y="264"
                textAnchor="middle"
                fill={isFixing ? "#2AD4C5" : "#78756C"}
                fontSize="11"
                fontFamily="monospace"
              >
                {isFixing
                  ? "REPLY 1 RECEIVED -> DRAFT FIX PR #48 -> REPLAY 6 HELD EVENTS"
                  : "WHATSAPP REPLY LOOP -> FIX PR + REPLAY HELD EVENTS"}
              </text>
            </svg>
          </div>

          {/* Synchronized narrative + interactive live WhatsApp preview */}
          <div className="mt-7 grid gap-6 lg:grid-cols-12 lg:items-center">
            <div className="lg:col-span-7">
              <div className="inline-block clip-hex-sm bg-basalt px-4 py-1 font-mono text-xs uppercase tracking-wider text-citron">
                {active.statusTag}
              </div>
              <h2 className="mt-3 font-display text-2xl font-bold text-chalk sm:text-3xl">
                {active.headline}
              </h2>
              <p className="mt-3 max-w-xl text-sm leading-relaxed text-ash sm:text-base">
                {active.body}
              </p>

              <div className="mt-5 flex flex-wrap items-center gap-3">
                {stepIndex < 3 ? (
                  <PrismButton
                    variant={stepIndex === 2 ? "cyan" : "citron"}
                    size="sm"
                    onClick={() => selectStep(stepIndex + 1)}
                  >
                    {stepIndex === 2 ? "Tap Reply 1 -> Draft Fix PR" : "Next step"}
                  </PrismButton>
                ) : (
                  <PrismButton
                    variant="olive"
                    size="sm"
                    onClick={() => selectStep(0)}
                  >
                    Run drill again
                  </PrismButton>
                )}
                <Link href="/deck">
                  <PrismButton variant="olive" size="sm">
                    Try full Control Deck
                  </PrismButton>
                </Link>
              </div>
            </div>

            {/* Chamfered WhatsApp live message vessel */}
            <div className="lg:col-span-5">
              <GeometricVessel
                shape="diamond-cut"
                tone={stepIndex >= 2 ? "cyan" : "moss"}
                innerClassName="p-4 sm:p-5 bg-silt"
              >
                <div className="flex items-center justify-between border-b border-moss pb-2.5">
                  <div className="flex items-center gap-2.5">
                    <svg width="16" height="16" viewBox="0 0 18 18" aria-hidden="true">
                      <circle cx="9" cy="9" r="7" fill="none" stroke="#2AD4C5" strokeWidth="1.5" />
                      <circle cx="9" cy="9" r="3" fill="#2AD4C5" />
                    </svg>
                    <span className="font-mono text-xs font-semibold uppercase tracking-wider text-chalk">
                      PockDev on WhatsApp
                    </span>
                  </div>
                  <span className="font-mono text-[10px] uppercase tracking-widest text-ash">
                    +234 809 441 2201
                  </span>
                </div>

                <div className="mt-3.5 space-y-2.5 font-sans text-xs leading-relaxed">
                  {stepIndex === 0 ? (
                    <div className="clip-bevel-chat bg-olive p-3.5 text-ash">
                      All quiet on Paystack Checkout. 148 webhooks delivered cleanly
                      today; PockDev stays silent until something breaks.
                    </div>
                  ) : (
                    <div className="clip-bevel-chat bg-olive p-3.5 text-chalk">
                      <p>
                        Paystack Checkout crashed on line 47 of{" "}
                        <span className="font-mono text-terracotta">
                          paystack.ts
                        </span>
                        . Paystack sent{" "}
                        <span className="font-mono text-cyan">
                          data.charged_amount
                        </span>{" "}
                        instead of <span className="font-mono">data.amount</span>.
                      </p>
                      <p className="mt-2 text-ash">
                        6 events are safely held. Reply 1 to draft a fix PR, 2 to
                        send held events again, or 3 to mute for 1h.
                      </p>
                    </div>
                  )}

                  {stepIndex === 3 ? (
                    <>
                      <div className="ml-auto max-w-[140px] clip-prism-sm bg-citron px-4 py-1.5 text-right font-mono text-xs font-bold text-silt">
                        1
                      </div>
                      <div className="clip-bevel-chat bg-basalt p-3.5 text-cyan">
                        Drafted Fix PR #48 on Mascottz/kora-checkout and replayed
                        all 6 held events (200 OK). All quiet now.
                      </div>
                    </>
                  ) : null}
                </div>
              </GeometricVessel>
            </div>
          </div>
        </GeometricVessel>
      </section>

      {/* Geometric SDK & Architecture Band (non-rectangular vessels) */}
      <section className="mt-16 grid gap-8 lg:grid-cols-12 lg:items-stretch">
        <div className="lg:col-span-5 flex flex-col justify-between gap-5">
          <GeometricVessel shape="octagon-sm" tone="moss" innerClassName="p-6">
            <div className="flex items-center gap-3">
              <svg width="24" height="24" viewBox="0 0 24 24" aria-hidden="true">
                <circle cx="12" cy="12" r="9" fill="none" stroke="#C9F23D" strokeWidth="1.8" />
                <circle cx="12" cy="12" r="3" fill="#C9F23D" />
              </svg>
              <h3 className="font-display text-lg font-bold text-chalk">
                One Shield URL per app
              </h3>
            </div>
            <p className="mt-2.5 text-sm leading-relaxed text-ash">
              Point Paystack, Stripe, or Flutterwave at your PockDev Shield URL.
              Every request is persisted before forwarding and flagged automatically
              when partner payload keys shift.
            </p>
          </GeometricVessel>

          <GeometricVessel shape="octagon-sm" tone="terracotta" innerClassName="p-6">
            <div className="flex items-center gap-3">
              <svg width="24" height="24" viewBox="0 0 24 24" aria-hidden="true">
                <polygon
                  points="12,3 22,20 2,20"
                  fill="none"
                  stroke="#E05A33"
                  strokeWidth="1.8"
                />
              </svg>
              <h3 className="font-display text-lg font-bold text-chalk">
                Exact crash-to-webhook linking
              </h3>
            </div>
            <p className="mt-2.5 text-sm leading-relaxed text-ash">
              Zero-dependency <span className="font-mono text-chalk">@pockdev/sdk</span>{" "}
              reads <span className="font-mono text-cyan">x-pockdev-delivery-id</span>{" "}
              automatically, tying the top stack frame directly to the held webhook
              so you fix root causes, not symptoms.
            </p>
          </GeometricVessel>
        </div>

        <div className="lg:col-span-7">
          <GeometricVessel
            shape="diamond-cut"
            tone="cyan"
            className="h-full"
            innerClassName="p-6 flex flex-col justify-between"
          >
            <div>
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <span className="font-mono text-[11px] uppercase tracking-[0.2em] text-cyan">
                    @pockdev/sdk: zero dependencies
                  </span>
                  <h3 className="mt-1 font-display text-xl font-bold text-chalk">
                    Wrap any webhook route in 3 lines
                  </h3>
                </div>
                <div className="flex items-center gap-1.5">
                  {(["nextjs", "express", "hono"] as SnippetFramework[]).map(
                    (fw) => (
                      <button
                        key={fw}
                        type="button"
                        onClick={() => setFramework(fw)}
                        className={`clip-prism-sm px-3 py-1 font-mono text-[11px] uppercase tracking-wider transition-colors ${
                          framework === fw
                            ? "bg-cyan text-silt font-bold"
                            : "bg-basalt text-ash hover:text-chalk"
                        }`}
                      >
                        {fw}
                      </button>
                    )
                  )}
                </div>
              </div>

              <div className="mt-4 clip-octagon-sm bg-silt p-4 border border-moss/60 overflow-x-auto">
                <pre className="font-mono text-xs leading-relaxed text-chalk">
                  <code>{SNIPPETS[framework]}</code>
                </pre>
              </div>
            </div>

            <div className="mt-5 flex flex-wrap items-center justify-between gap-3 pt-2">
              <span className="font-mono text-xs text-ash">
                Posts directly to POST /v1/errors with X-PockDev-Key
              </span>
              <PrismButton variant="cyan" size="sm" onClick={handleCopySnippet}>
                {copied ? "Copied snippet" : "Copy SDK snippet"}
              </PrismButton>
            </div>
          </GeometricVessel>
        </div>
      </section>
    </div>
  );
}
