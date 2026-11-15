"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { slugifyName } from "@/lib/demo-data";
import {
  formatNairaFromKobo,
  isQuotaExceededError,
} from "@/lib/engine-client";
import type { BillingStatus, CreateProjectInput, Project } from "@/lib/types";
import { GeometricVessel } from "./GeometricVessel";
import { PrismButton } from "./PrismButton";

interface NewProjectModalProps {
  open: boolean;
  engineUrl: string;
  billing?: BillingStatus | null;
  upgrading?: boolean;
  onUpgrade?: () => Promise<void> | void;
  onClose: () => void;
  onCreate: (input: CreateProjectInput) => Promise<Project>;
}

type SdkTarget = "nextjs" | "express" | "fastify" | "hono" | "node";

function buildSdkSnippet(
  target: SdkTarget,
  sdkKey: string,
  endpoint: string
): string {
  const cleanEndpoint = endpoint.replace(/\/+$/, "");
  const header = `import { init, wrap } from "@pockdev/sdk";\n\ninit({\n  key: "${sdkKey}",\n  endpoint: "${cleanEndpoint}",\n});\n`;

  switch (target) {
    case "nextjs":
      return `${header}\nexport const POST = wrap(async (req: Request) => {\n  const payload = await req.json();\n  // Handle verified webhook event here\n  return Response.json({ received: true, event: payload.event });\n});`;
    case "express":
      return `${header}\napp.post("/webhooks", wrap(async (req, res) => {\n  // Handle verified webhook event here\n  res.json({ received: true });\n}));`;
    case "fastify":
      return `${header}\nfastify.post("/webhooks", wrap(async (request, reply) => {\n  // Handle verified webhook event here\n  return reply.send({ received: true });\n}));`;
    case "hono":
      return `${header}\napp.post("/webhooks", wrap(async (c) => {\n  const payload = await c.req.json();\n  return c.json({ received: true, event: payload.event });\n}));`;
    case "node":
      return `${header}\nimport http from "node:http";\n\nhttp.createServer(wrap(async (req, res) => {\n  res.writeHead(200, { "Content-Type": "application/json" });\n  res.end(JSON.stringify({ received: true }));\n})).listen(3001);`;
  }
}

export function NewProjectModal({
  open,
  engineUrl,
  billing = null,
  upgrading = false,
  onUpgrade,
  onClose,
  onCreate,
}: NewProjectModalProps) {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugTouched, setSlugTouched] = useState(false);
  const [targetUrl, setTargetUrl] = useState("");
  const [githubRepo, setGithubRepo] = useState("");
  const [whatsappPhone, setWhatsappPhone] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [quotaErrorMessage, setQuotaErrorMessage] = useState<string | null>(
    null
  );
  const [created, setCreated] = useState<Project | null>(null);
  const [sdkTarget, setSdkTarget] = useState<SdkTarget>("nextjs");
  const [copiedKey, setCopiedKey] = useState<"url" | "sdk" | "snippet" | null>(
    null
  );

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  const limitReachedFromBilling = Boolean(
    billing &&
      billing.usage.projectsLimit > 0 &&
      billing.usage.projectsUsed >= billing.usage.projectsLimit
  );
  const showQuotaUpgrade = !created && (limitReachedFromBilling || Boolean(quotaErrorMessage));
  const proPriceLabel = billing
    ? formatNairaFromKobo(billing.proPriceKobo, billing.currency)
    : "NGN 15,000";

  const resetAndClose = () => {
    setName("");
    setSlug("");
    setSlugTouched(false);
    setTargetUrl("");
    setGithubRepo("");
    setWhatsappPhone("");
    setError("");
    setQuotaErrorMessage(null);
    setCreated(null);
    onClose();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError("Please enter an app name.");
      return;
    }
    setSubmitting(true);
    setError("");
    setQuotaErrorMessage(null);
    try {
      const project = await onCreate({
        name: name.trim(),
        slug: (slug.trim() || slugifyName(name)).trim(),
        targetUrl: targetUrl.trim(),
        githubRepo: githubRepo.trim(),
        whatsappPhone: whatsappPhone.trim(),
      });
      setCreated(project);
    } catch (err) {
      if (isQuotaExceededError(err)) {
        setQuotaErrorMessage(
          err instanceof Error
            ? err.message
            : "Free plan includes up to 2 watched apps. Upgrade to Pro for up to 20 apps."
        );
      } else {
        setError(
          err instanceof Error ? err.message : "Could not create project."
        );
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleUpgradeClick = async () => {
    if (!onUpgrade) return;
    setError("");
    try {
      await onUpgrade();
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Could not launch Paystack checkout."
      );
    }
  };

  const copyText = async (
    value: string,
    slot: "url" | "sdk" | "snippet"
  ) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedKey(slot);
      window.setTimeout(() => setCopiedKey(null), 1800);
    } catch {
      // Ignore clipboard errors in restricted browsers.
    }
  };

  const cleanEngine = engineUrl.replace(/\/+$/, "");
  const shieldUrl = created ? `${cleanEngine}/in/${created.endpointKey}` : "";
  const snippet = created
    ? buildSdkSnippet(sdkTarget, created.sdkKey, cleanEngine)
    : "";

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-silt/85 p-4 backdrop-blur-sm overflow-y-auto"
      role="dialog"
      aria-modal="true"
      aria-label="Connect a new app to PockDev"
    >
      <div className="w-full max-w-2xl my-auto">
        <GeometricVessel
          shape="diamond-cut"
          tone={created ? "cyan" : showQuotaUpgrade ? "terracotta" : "citron"}
          innerClassName="p-6 sm:p-8"
        >
          <div className="flex items-start justify-between gap-4 border-b border-moss pb-4">
            <div>
              <span
                className={`font-mono text-[11px] uppercase tracking-[0.2em] ${
                  showQuotaUpgrade ? "text-terracotta" : "text-citron"
                }`}
              >
                {created
                  ? "Shield Ready"
                  : showQuotaUpgrade
                    ? "Plan Capacity Reached"
                    : "New Orbital Node"}
              </span>
              <h2 className="mt-1 font-display text-2xl font-extrabold text-chalk">
                {created
                  ? `${created.name} is now shielded`
                  : showQuotaUpgrade
                    ? "Upgrade to add more apps"
                    : "Add an app to PockDev"}
              </h2>
            </div>
            <PrismButton variant="olive" size="sm" onClick={resetAndClose}>
              Close
            </PrismButton>
          </div>

          {showQuotaUpgrade ? (
            <div className="mt-6 space-y-5">
              <GeometricVessel
                shape="octagon-sm"
                tone="citron"
                innerClassName="bg-silt p-5 sm:p-6"
              >
                <div className="flex items-start gap-4">
                  <svg
                    width="44"
                    height="44"
                    viewBox="0 0 48 48"
                    className="shrink-0"
                    aria-hidden="true"
                  >
                    <polygon
                      points="24,4 44,15 44,33 24,44 4,33 4,15"
                      fill="#141813"
                      stroke="#E05A33"
                      strokeWidth="1.8"
                    />
                    <polygon points="24,14 34,31 14,31" fill="none" stroke="#C9F23D" strokeWidth="1.8" />
                    <circle cx="24" cy="25" r="2.2" fill="#C9F23D" />
                  </svg>
                  <div>
                    <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-citron">
                      {billing?.plan === "pro" ? "Pro Orbit Limit" : "Free Shield Cap"}
                    </p>
                    <p className="mt-1.5 text-sm leading-relaxed text-chalk">
                      {quotaErrorMessage ||
                        `This workspace is watching ${billing?.usage.projectsUsed ?? 2} of ${billing?.usage.projectsLimit ?? 2} allowed apps on Free Shield. Upgrade to Pro Orbit to watch up to 20 apps, capture 100,000 monthly events, and keep 30 days of replay history.`}
                    </p>
                  </div>
                </div>

                <div className="mt-5 grid grid-cols-3 gap-2.5">
                  <div className="clip-octagon-sm bg-basalt px-3 py-2.5 text-center">
                    <p className="font-display text-base font-extrabold text-citron">
                      20 Apps
                    </p>
                    <p className="font-mono text-[10px] uppercase tracking-wider text-ash">
                      Up from 2
                    </p>
                  </div>
                  <div className="clip-octagon-sm bg-basalt px-3 py-2.5 text-center">
                    <p className="font-display text-base font-extrabold text-citron">
                      100k / mo
                    </p>
                    <p className="font-mono text-[10px] uppercase tracking-wider text-ash">
                      Events
                    </p>
                  </div>
                  <div className="clip-octagon-sm bg-basalt px-3 py-2.5 text-center">
                    <p className="font-display text-base font-extrabold text-citron">
                      30 Days
                    </p>
                    <p className="font-mono text-[10px] uppercase tracking-wider text-ash">
                      Retention
                    </p>
                  </div>
                </div>
              </GeometricVessel>

              {error ? (
                <div className="clip-prism-sm bg-terracotta/20 border-l-2 border-terracotta px-4 py-2 font-mono text-xs text-terracotta">
                  {error}
                </div>
              ) : null}

              <div className="flex flex-wrap items-center justify-between gap-3 pt-1">
                <Link
                  href="/pricing"
                  onClick={resetAndClose}
                  className="clip-prism-sm bg-basalt px-4 py-2 font-mono text-[11px] uppercase tracking-[0.16em] text-ash transition-colors hover:bg-moss hover:text-chalk"
                >
                  Compare plans
                </Link>

                <div className="flex flex-wrap items-center gap-3">
                  <PrismButton
                    type="button"
                    variant="olive"
                    size="md"
                    onClick={resetAndClose}
                  >
                    Not now
                  </PrismButton>
                  {onUpgrade && billing?.plan !== "pro" ? (
                    <PrismButton
                      type="button"
                      variant="citron"
                      size="md"
                      disabled={upgrading}
                      onClick={handleUpgradeClick}
                    >
                      {upgrading
                        ? "Opening Paystack..."
                        : `Upgrade to Pro (${proPriceLabel}/mo)`}
                    </PrismButton>
                  ) : null}
                </div>
              </div>
            </div>
          ) : !created ? (
            <form onSubmit={handleSubmit} className="mt-6 space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <label className="block">
                  <span className="font-mono text-[11px] uppercase tracking-wider text-ash">
                    App name *
                  </span>
                  <input
                    type="text"
                    required
                    placeholder="Checkout API"
                    value={name}
                    onChange={(e) => {
                      const val = e.target.value;
                      setName(val);
                      if (!slugTouched) setSlug(slugifyName(val));
                    }}
                    className="mt-1.5 w-full clip-octagon-sm bg-silt px-4 py-2.5 text-sm text-chalk border border-moss focus:border-citron focus:outline-none"
                  />
                </label>

                <label className="block">
                  <span className="font-mono text-[11px] uppercase tracking-wider text-ash">
                    Slug
                  </span>
                  <input
                    type="text"
                    placeholder="checkout-api"
                    value={slug}
                    onChange={(e) => {
                      setSlugTouched(true);
                      setSlug(e.target.value);
                    }}
                    className="mt-1.5 w-full clip-octagon-sm bg-silt px-4 py-2.5 font-mono text-sm text-chalk border border-moss focus:border-citron focus:outline-none"
                  />
                </label>
              </div>

              <label className="block">
                <span className="font-mono text-[11px] uppercase tracking-wider text-ash">
                  Target webhook URL
                </span>
                <input
                  type="url"
                  placeholder="https://api.myapp.com/webhooks/paystack"
                  value={targetUrl}
                  onChange={(e) => setTargetUrl(e.target.value)}
                  className="mt-1.5 w-full clip-octagon-sm bg-silt px-4 py-2.5 font-mono text-sm text-chalk border border-moss focus:border-citron focus:outline-none"
                />
              </label>

              <div className="grid gap-4 sm:grid-cols-2">
                <label className="block">
                  <span className="font-mono text-[11px] uppercase tracking-wider text-ash">
                    GitHub repo (owner/name)
                  </span>
                  <input
                    type="text"
                    placeholder="Mascottz/checkout-api"
                    value={githubRepo}
                    onChange={(e) => setGithubRepo(e.target.value)}
                    className="mt-1.5 w-full clip-octagon-sm bg-silt px-4 py-2.5 font-mono text-sm text-chalk border border-moss focus:border-citron focus:outline-none"
                  />
                </label>

                <label className="block">
                  <span className="font-mono text-[11px] uppercase tracking-wider text-ash">
                    WhatsApp phone
                  </span>
                  <input
                    type="tel"
                    placeholder="+2348094412201"
                    value={whatsappPhone}
                    onChange={(e) => setWhatsappPhone(e.target.value)}
                    className="mt-1.5 w-full clip-octagon-sm bg-silt px-4 py-2.5 font-mono text-sm text-chalk border border-moss focus:border-citron focus:outline-none"
                  />
                </label>
              </div>

              {error ? (
                <div className="clip-prism-sm bg-terracotta/20 border-l-2 border-terracotta px-4 py-2 font-mono text-xs text-terracotta">
                  {error}
                </div>
              ) : null}

              <div className="flex flex-wrap items-center justify-end gap-3 pt-3">
                <PrismButton
                  type="button"
                  variant="olive"
                  size="md"
                  onClick={resetAndClose}
                >
                  Cancel
                </PrismButton>
                <PrismButton
                  type="submit"
                  variant="citron"
                  size="md"
                  disabled={submitting}
                >
                  {submitting ? "Creating shield..." : "Create Shield Node"}
                </PrismButton>
              </div>
            </form>
          ) : (
            <div className="mt-6 space-y-5">
              {/* Step 1: Shield URL */}
              <GeometricVessel
                shape="octagon-sm"
                tone="citron"
                innerClassName="p-4 bg-silt"
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="font-mono text-[11px] uppercase tracking-wider text-citron">
                    1. Paste this Shield URL into your webhook provider
                  </span>
                  <PrismButton
                    variant="citron"
                    size="sm"
                    onClick={() => copyText(shieldUrl, "url")}
                  >
                    {copiedKey === "url" ? "Copied URL" : "Copy Shield URL"}
                  </PrismButton>
                </div>
                <p className="mt-2.5 break-all font-mono text-xs text-chalk">
                  {shieldUrl}
                </p>
              </GeometricVessel>

              {/* Step 2: SDK Key */}
              <GeometricVessel
                shape="octagon-sm"
                tone="moss"
                innerClassName="p-4 bg-silt"
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="font-mono text-[11px] uppercase tracking-wider text-ash">
                    2. SDK Key (X-PockDev-Key)
                  </span>
                  <PrismButton
                    variant="olive"
                    size="sm"
                    onClick={() => copyText(created.sdkKey, "sdk")}
                  >
                    {copiedKey === "sdk" ? "Copied Key" : "Copy SDK Key"}
                  </PrismButton>
                </div>
                <p className="mt-2 break-all font-mono text-xs text-cyan">
                  {created.sdkKey}
                </p>
              </GeometricVessel>

              {/* Step 3: Copyable @pockdev/sdk Snippet */}
              <GeometricVessel
                shape="octagon-sm"
                tone="cyan"
                innerClassName="p-4 bg-silt"
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="font-mono text-[11px] uppercase tracking-wider text-cyan">
                    3. Wrap your route with @pockdev/sdk
                  </span>
                  <div className="flex flex-wrap items-center gap-1.5">
                    {(
                      ["nextjs", "express", "fastify", "hono", "node"] as SdkTarget[]
                    ).map((fw) => (
                      <button
                        key={fw}
                        type="button"
                        onClick={() => setSdkTarget(fw)}
                        className={`clip-prism-sm px-2.5 py-0.5 font-mono text-[10px] uppercase tracking-wider ${
                          sdkTarget === fw
                            ? "bg-cyan text-silt font-bold"
                            : "bg-basalt text-ash hover:text-chalk"
                        }`}
                      >
                        {fw}
                      </button>
                    ))}
                  </div>
                </div>

                <pre className="mt-3 overflow-x-auto font-mono text-xs leading-relaxed text-chalk">
                  <code>{snippet}</code>
                </pre>

                <div className="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-moss/60 pt-3">
                  <span className="font-mono text-[11px] text-ash">
                    npm install @pockdev/sdk
                  </span>
                  <PrismButton
                    variant="cyan"
                    size="sm"
                    onClick={() => copyText(snippet, "snippet")}
                  >
                    {copiedKey === "snippet"
                      ? "Copied snippet"
                      : "Copy SDK snippet"}
                  </PrismButton>
                </div>
              </GeometricVessel>

              <div className="flex justify-end">
                <PrismButton variant="citron" size="md" onClick={resetAndClose}>
                  Back to Control Deck
                </PrismButton>
              </div>
            </div>
          )}
        </GeometricVessel>
      </div>
    </div>
  );
}
