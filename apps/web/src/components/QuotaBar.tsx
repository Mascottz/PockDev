"use client";

import Link from "next/link";
import { GeometricVessel } from "./GeometricVessel";
import { PrismButton } from "./PrismButton";
import { formatNairaFromKobo } from "@/lib/engine-client";
import type { BillingStatus } from "@/lib/types";

interface QuotaBarProps {
  billing: BillingStatus;
  upgrading: boolean;
  onUpgrade: () => void;
}

function formatCompactCount(value: number): string {
  if (value >= 1000000) {
    return `${(value / 1000000).toFixed(1).replace(/\.0$/, "")}M`;
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(value % 1000 === 0 ? 0 : 1)}k`;
  }
  return value.toLocaleString("en-US");
}

function formatExpiryDate(iso: string | null): string | null {
  if (!iso) return null;
  const parsed = Date.parse(iso);
  if (Number.isNaN(parsed)) return null;
  return new Date(parsed).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

interface RadialGaugeProps {
  label: string;
  valueText: string;
  ratio: number;
  accent: string;
  subText: string;
}

function RadialGauge({
  label,
  valueText,
  ratio,
  accent,
  subText,
}: RadialGaugeProps) {
  const clamped = Math.max(0, Math.min(1, Number.isFinite(ratio) ? ratio : 0));
  const radius = 21;
  const circumference = 2 * Math.PI * radius;
  const dashOffset = circumference * (1 - clamped);

  return (
    <div className="flex items-center gap-3">
      <div className="relative h-14 w-14 shrink-0">
        <svg
          width="56"
          height="56"
          viewBox="0 0 56 56"
          className="overflow-visible"
          aria-hidden="true"
        >
          <circle
            cx="28"
            cy="28"
            r="26"
            fill="none"
            stroke="#363F30"
            strokeWidth="1"
            strokeDasharray="2 4"
          />
          <circle
            cx="28"
            cy="28"
            r={radius}
            fill="#141813"
            stroke="#2B3327"
            strokeWidth="3.5"
          />
          <circle
            cx="28"
            cy="28"
            r={radius}
            fill="none"
            stroke={accent}
            strokeWidth="3.5"
            strokeLinecap="butt"
            strokeDasharray={circumference}
            strokeDashoffset={dashOffset}
            transform="rotate(-90 28 28)"
          />
          <text
            x="28"
            y="31"
            textAnchor="middle"
            fill="#F3EFE6"
            fontSize="10"
            fontWeight="700"
            className="font-mono"
          >
            {Math.round(clamped * 100)}%
          </text>
        </svg>
      </div>
      <div className="min-w-0">
        <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-ash">
          {label}
        </p>
        <p className="mt-0.5 font-display text-sm font-bold text-chalk">
          {valueText}
        </p>
        <p className="font-mono text-[10px] text-ash">{subText}</p>
      </div>
    </div>
  );
}

export function QuotaBar({ billing, upgrading, onUpgrade }: QuotaBarProps) {
  const isPro = billing.plan === "pro";
  const { usage } = billing;

  const projectsRatio =
    usage.projectsLimit > 0 ? usage.projectsUsed / usage.projectsLimit : 0;
  const eventsRatio =
    usage.monthlyEventsLimit > 0
      ? usage.monthlyEventsUsed / usage.monthlyEventsLimit
      : 0;
  const retentionRatio = Math.min(1, usage.retentionDays / 30);

  const atProjectLimit =
    usage.projectsLimit > 0 && usage.projectsUsed >= usage.projectsLimit;
  const nearEventLimit = eventsRatio >= 0.85;

  const projectsAccent = atProjectLimit
    ? "#E05A33"
    : projectsRatio >= 0.75
      ? "#C9F23D"
      : "#2AD4C5";
  const eventsAccent = nearEventLimit
    ? "#E05A33"
    : isPro
      ? "#C9F23D"
      : "#2AD4C5";
  const retentionAccent = isPro ? "#C9F23D" : "#2AD4C5";

  const expiryText = formatExpiryDate(billing.planExpiresAt);
  const proPriceText = formatNairaFromKobo(
    billing.proPriceKobo,
    billing.currency
  );

  return (
    <GeometricVessel
      shape="diamond-cut"
      tone={isPro ? "citron" : atProjectLimit ? "terracotta" : "moss"}
      innerClassName="px-5 py-4 sm:px-6"
    >
      <div className="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
        {/* Plan Identity */}
        <div className="flex items-start gap-3.5">
          <div
            className={`clip-hex-sm flex h-11 w-11 shrink-0 items-center justify-center ${
              isPro ? "bg-citron/15" : "bg-basalt"
            }`}
          >
            {isPro ? (
              <svg width="24" height="24" viewBox="0 0 24 24" aria-hidden="true">
                <circle
                  cx="12"
                  cy="12"
                  r="9"
                  fill="none"
                  stroke="#C9F23D"
                  strokeWidth="1.5"
                  strokeDasharray="6 3"
                />
                <polygon points="12,5 17,12 12,19 7,12" fill="#C9F23D" />
              </svg>
            ) : (
              <svg width="24" height="24" viewBox="0 0 24 24" aria-hidden="true">
                <polygon
                  points="12,3 20,7 20,14 12,21 4,14 4,7"
                  fill="none"
                  stroke={atProjectLimit ? "#E05A33" : "#2AD4C5"}
                  strokeWidth="1.7"
                />
                <polygon
                  points="12,8 16,14 8,14"
                  fill={atProjectLimit ? "#E05A33" : "#2AD4C5"}
                />
              </svg>
            )}
          </div>

          <div>
            <div className="flex flex-wrap items-center gap-2">
              <span
                className={`clip-hex-sm px-2.5 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.2em] ${
                  isPro
                    ? "bg-citron text-silt"
                    : atProjectLimit
                      ? "bg-terracotta/20 text-terracotta"
                      : "bg-basalt text-chalk"
                }`}
              >
                {isPro ? "PRO ORBIT" : "FREE SHIELD"}
              </span>
              <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-ash">
                {billing.planStatus || (isPro ? "active" : "free")}
              </span>
            </div>

            <p className="mt-1.5 text-xs text-ash">
              {isPro
                ? expiryText
                  ? `Active through ${expiryText}. 20 apps and 100k events/mo unlocked.`
                  : "Active Pro workspace: 20 watched apps and 100k monthly events unlocked."
                : atProjectLimit
                  ? `App limit reached (${usage.projectsUsed}/${usage.projectsLimit}). Upgrade to watch up to 20 apps.`
                  : "Instant Paystack upgrade expands this workspace to 20 apps and 30-day history."}
            </p>
          </div>
        </div>

        {/* Radial Usage Gauges */}
        <div className="grid grid-cols-1 gap-4 border-y border-moss/60 py-3.5 sm:grid-cols-3 lg:border-y-0 lg:border-x lg:px-6 lg:py-0">
          <RadialGauge
            label="Apps"
            valueText={`${usage.projectsUsed} / ${usage.projectsLimit}`}
            ratio={projectsRatio}
            accent={projectsAccent}
            subText={atProjectLimit ? "At cap" : "Watched"}
          />
          <RadialGauge
            label="Monthly Events"
            valueText={`${formatCompactCount(usage.monthlyEventsUsed)} / ${formatCompactCount(usage.monthlyEventsLimit)}`}
            ratio={eventsRatio}
            accent={eventsAccent}
            subText="This cycle"
          />
          <RadialGauge
            label="Retention"
            valueText={`${usage.retentionDays} days`}
            ratio={retentionRatio}
            accent={retentionAccent}
            subText={isPro ? "Full 30d window" : "3d Free window"}
          />
        </div>

        {/* Upgrade / Plan Action */}
        <div className="flex flex-wrap items-center gap-3 lg:flex-col lg:items-end lg:gap-1.5">
          {isPro ? (
            <>
              <span className="clip-prism-sm bg-citron/15 px-3.5 py-1.5 font-mono text-[11px] font-semibold uppercase tracking-[0.18em] text-citron">
                Pro Orbit Active
              </span>
              <Link
                href="/pricing"
                className="font-mono text-[10px] uppercase tracking-[0.18em] text-ash transition-colors hover:text-citron"
              >
                Compare plans -&gt;
              </Link>
            </>
          ) : (
            <>
              <PrismButton
                type="button"
                variant="citron"
                size="sm"
                disabled={upgrading}
                onClick={onUpgrade}
              >
                {upgrading ? "Opening Paystack..." : "Upgrade to Pro"}
              </PrismButton>
              <div className="flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.16em] text-ash">
                <span>{proPriceText}/mo</span>
                <span className="text-moss">/</span>
                <Link
                  href="/pricing"
                  className="text-cyan transition-colors hover:text-citron"
                >
                  Compare
                </Link>
              </div>
            </>
          )}
        </div>
      </div>
    </GeometricVessel>
  );
}
