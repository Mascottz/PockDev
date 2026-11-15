import type { Metadata } from "next";
import { BrandMark } from "@/components/BrandMark";
import { GeometricVessel } from "@/components/GeometricVessel";

export const metadata: Metadata = {
  title: "Brand Kit",
  description:
    "Official PockDev geometric marks, lockups, color swatches, and typefaces.",
};

const SWATCHES: { name: string; hex: string; role: string; textDark: boolean }[] = [
  { name: "Silt", hex: "#0C0E0B", role: "Command surface", textDark: false },
  { name: "Olive", hex: "#151913", role: "Vessel interior", textDark: false },
  { name: "Citron", hex: "#C9F23D", role: "Signal and action", textDark: true },
  { name: "Terracotta", hex: "#E05A33", role: "Crash and beacon", textDark: true },
  { name: "Cyan", hex: "#2AD4C5", role: "Core status", textDark: true },
  { name: "Chalk", hex: "#F3EFE6", role: "Light canvas", textDark: true },
];

function DownloadLink({ href, label }: { href: string; label: string }) {
  return (
    <a
      href={href}
      download
      className="clip-prism inline-block bg-citron/60 p-[1px] transition-transform active:scale-[0.98]"
    >
      <span className="clip-prism inline-flex items-center gap-2.5 bg-basalt px-5 py-2 text-xs font-bold uppercase tracking-wider text-citron">
        <svg width="11" height="11" viewBox="0 0 12 12" aria-hidden="true">
          <polygon points="6,11 11,1 1,1" fill="currentColor" />
        </svg>
        {label}
      </span>
    </a>
  );
}

function SectionHeading({ kicker, title }: { kicker: string; title: string }) {
  return (
    <div className="mb-6 flex items-center gap-3">
      <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
        <polygon points="6,1 11,10 1,10" fill="#C9F23D" />
      </svg>
      <div>
        <p className="font-mono text-[10px] uppercase tracking-[0.28em] text-ash">{kicker}</p>
        <h2 className="font-display text-2xl font-extrabold tracking-tight text-chalk">{title}</h2>
      </div>
    </div>
  );
}

export default function BrandPage() {
  return (
    <div className="mx-auto max-w-7xl px-5 py-14 sm:px-8">
      {/* Hero */}
      <div className="relative overflow-hidden">
        <div className="clip-octagon bg-moss p-[1px]">
          <div className="clip-octagon bg-olive">
            <div className="flex flex-col gap-8 px-8 py-12 sm:flex-row sm:items-center sm:justify-between sm:px-14">
              <div>
                <p className="font-mono text-[11px] uppercase tracking-[0.32em] text-citron">
                  Webhook Shield &amp; Pocket SRE
                </p>
                <h1 className="mt-3 font-display text-5xl font-extrabold tracking-tight text-chalk sm:text-6xl">
                  Pock<span className="text-citron">Dev</span> Brand Kit
                </h1>
                <p className="mt-4 max-w-xl text-sm leading-relaxed text-ash">
                  Official geometric marks, lockups, color swatches, and typefaces. Every asset is
                  built from chamfered polygons, orbital rings, and triangular beacons. Download the
                  SVGs and ship with them.
                </p>
              </div>
              <div className="flex items-center gap-6">
                <BrandMark size={140} animated />
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Marks and lockups */}
      <section className="mt-14" aria-label="Marks and lockups">
        <SectionHeading kicker="Identity" title="Marks and Lockups" />
        <div className="grid gap-6 lg:grid-cols-2">
          <GeometricVessel shape="diamond-cut" tone="moss" innerClassName="bg-olive">
            <div className="flex flex-col items-center gap-6 px-6 py-10">
              <BrandMark size={216} animated />
              <div className="text-center">
                <p className="font-display text-xl font-extrabold text-chalk">Primary Octagon Mark</p>
                <p className="mt-1 font-mono text-[11px] uppercase tracking-[0.2em] text-ash">
                  Shield, orbital ring, delta beacon, cyan core
                </p>
              </div>
              <DownloadLink href="/brand/pockdev-mark.svg" label="Download SVG" />
            </div>
          </GeometricVessel>

          <div className="flex flex-col gap-6">
            <GeometricVessel shape="hex-bar" tone="moss" innerClassName="bg-silt">
              <div className="flex flex-col items-center gap-5 px-6 py-7">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src="/brand/pockdev-wordmark-dark.svg"
                  alt="PockDev dark lockup"
                  className="w-full max-w-[440px]"
                />
                <div className="flex items-center gap-4">
                  <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-ash">
                    Dark Lockup
                  </p>
                  <DownloadLink href="/brand/pockdev-wordmark-dark.svg" label="Download SVG" />
                </div>
              </div>
            </GeometricVessel>

            <GeometricVessel shape="hex-bar" tone="chalk" innerClassName="bg-chalk">
              <div className="flex flex-col items-center gap-5 px-6 py-7">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src="/brand/pockdev-wordmark-light.svg"
                  alt="PockDev chalk light lockup"
                  className="w-full max-w-[440px]"
                />
                <div className="flex items-center gap-4">
                  <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-silt/70">
                    Chalk Light Lockup
                  </p>
                  <DownloadLink href="/brand/pockdev-wordmark-light.svg" label="Download SVG" />
                </div>
              </div>
            </GeometricVessel>
          </div>
        </div>
      </section>

      {/* Swatches */}
      <section className="mt-16" aria-label="Color swatches">
        <SectionHeading kicker="Palette" title="Official Color Swatches" />
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {SWATCHES.map((swatch) => (
            <GeometricVessel key={swatch.hex} shape="prism" tone="moss" innerClassName="bg-olive">
              <div className="flex flex-col">
                <div
                  className="clip-prism-sm flex h-36 items-end p-4"
                  style={{ backgroundColor: swatch.hex }}
                >
                  <span
                    className={`clip-prism-sm px-3 py-1 font-mono text-[11px] font-medium tracking-[0.16em] ${
                      swatch.textDark ? "bg-silt/85 text-chalk" : "bg-chalk/90 text-silt"
                    }`}
                  >
                    {swatch.hex}
                  </span>
                </div>
                <div className="flex items-center justify-between gap-3 px-5 py-4">
                  <div>
                    <p className="font-display text-lg font-extrabold text-chalk">{swatch.name}</p>
                    <p className="mt-0.5 font-mono text-[10px] uppercase tracking-[0.18em] text-ash">
                      {swatch.role}
                    </p>
                  </div>
                  <svg width="18" height="18" viewBox="0 0 12 12" aria-hidden="true">
                    <polygon points="6,1 11,10 1,10" fill={swatch.hex} />
                  </svg>
                </div>
              </div>
            </GeometricVessel>
          ))}
        </div>
      </section>

      {/* Type */}
      <section className="mt-16" aria-label="Typefaces">
        <SectionHeading kicker="Voice" title="Typefaces" />
        <div className="grid gap-6 lg:grid-cols-3">
          <GeometricVessel shape="octagon-sm" tone="moss" innerClassName="bg-olive">
            <div className="px-8 py-9">
              <p className="font-mono text-[10px] uppercase tracking-[0.24em] text-citron">
                Display
              </p>
              <p className="mt-3 font-display text-4xl font-extrabold tracking-tight text-chalk">
                Syne
              </p>
              <p className="mt-1 font-display text-sm text-ash">Syne Variable, weights 400 to 800</p>
              <p className="mt-5 font-display text-xl font-bold text-chalk">
                Catch every webhook.
              </p>
              <p className="mt-2 text-xs leading-relaxed text-ash">
                Headlines, product titles, and mark lockups. Angular voice that matches the octagon
                geometry.
              </p>
            </div>
          </GeometricVessel>

          <GeometricVessel shape="octagon-sm" tone="moss" innerClassName="bg-olive">
            <div className="px-8 py-9">
              <p className="font-mono text-[10px] uppercase tracking-[0.24em] text-citron">
                Interface
              </p>
              <p className="mt-3 text-4xl font-extrabold tracking-tight text-chalk">
                Plus Jakarta Sans
              </p>
              <p className="mt-1 text-sm text-ash">Plus Jakarta Sans Variable, clean UI copy</p>
              <p className="mt-5 text-xl font-bold text-chalk">Fix crashes on WhatsApp.</p>
              <p className="mt-2 text-xs leading-relaxed text-ash">
                Body copy, buttons, and interface labels. Friendly geometry without losing the
                engineering tone.
              </p>
            </div>
          </GeometricVessel>

          <GeometricVessel shape="octagon-sm" tone="moss" innerClassName="bg-olive">
            <div className="px-8 py-9">
              <p className="font-mono text-[10px] uppercase tracking-[0.24em] text-citron">
                Telemetry
              </p>
              <p className="mt-3 font-mono text-4xl font-semibold tracking-tight text-chalk">
                IBM Plex Mono
              </p>
              <p className="mt-1 font-mono text-sm text-ash">IBM Plex Mono, weights 400 to 600</p>
              <p className="mt-5 font-mono text-xl text-chalk">webhook.held: replay queued</p>
              <p className="mt-2 text-xs leading-relaxed text-ash">
                Logs, stack lines, hashes, and metadata. Monospaced truth for incident work.
              </p>
            </div>
          </GeometricVessel>
        </div>
      </section>
    </div>
  );
}
