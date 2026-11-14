"use client";

import { useEffect, useState } from "react";
import { slugifyName } from "@/lib/demo-data";
import type { CreateProjectInput, Project } from "@/lib/types";
import { GeometricVessel } from "./GeometricVessel";
import { PrismButton } from "./PrismButton";

interface NewProjectModalProps {
  open: boolean;
  engineUrl: string;
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

  const resetAndClose = () => {
    setName("");
    setSlug("");
    setSlugTouched(false);
    setTargetUrl("");
    setGithubRepo("");
    setWhatsappPhone("");
    setError("");
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
      setError(
        err instanceof Error ? err.message : "Could not create project."
      );
    } finally {
      setSubmitting(false);
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
          tone={created ? "cyan" : "citron"}
          innerClassName="p-6 sm:p-8"
        >
          <div className="flex items-start justify-between gap-4 border-b border-moss pb-4">
            <div>
              <span className="font-mono text-[11px] uppercase tracking-[0.2em] text-citron">
                {created ? "Shield Ready" : "New Orbital Node"}
              </span>
              <h2 className="mt-1 font-display text-2xl font-extrabold text-chalk">
                {created
                  ? `${created.name} is now shielded`
                  : "Add an app to PockDev"}
              </h2>
            </div>
            <PrismButton variant="olive" size="sm" onClick={resetAndClose}>
              Close
            </PrismButton>
          </div>

          {!created ? (
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
