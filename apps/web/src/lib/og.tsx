import fs from "node:fs";
import path from "node:path";

/**
 * Shared geometry and font loading for the next/og image routes. Every shape
 * is a filled bezier path so satori renders it exactly like the brand SVGs.
 */

export const PALETTE = {
  silt: "#0C0E0B",
  olive: "#151913",
  basalt: "#1E241B",
  moss: "#2C3527",
  citron: "#C9F23D",
  terracotta: "#E05A33",
  cyan: "#2AD4C5",
  chalk: "#F3EFE6",
  ash: "#A5A196",
} as const;

const TAU = Math.PI * 2;

function pt(cx: number, cy: number, r: number, t: number): [number, number] {
  return [cx + r * Math.cos(t), cy + r * Math.sin(t)];
}

function bezierArc(cx: number, cy: number, r: number, t1: number, t2: number): string {
  const span = t2 - t1;
  const chunks = Math.max(1, Math.ceil(Math.abs(span) / (Math.PI / 2)));
  const step = span / chunks;
  let d = "";
  for (let i = 0; i < chunks; i += 1) {
    const a1 = t1 + step * i;
    const a2 = a1 + step;
    const k = (4 / 3) * Math.tan((a2 - a1) / 4);
    const [x1, y1] = pt(cx, cy, r, a1);
    const [x2, y2] = pt(cx, cy, r, a2);
    const c1x = x1 - k * r * Math.sin(a1);
    const c1y = y1 + k * r * Math.cos(a1);
    const c2x = x2 + k * r * Math.sin(a2);
    const c2y = y2 - k * r * Math.cos(a2);
    d += ` C ${c1x.toFixed(2)} ${c1y.toFixed(2)} ${c2x.toFixed(2)} ${c2y.toFixed(2)} ${x2.toFixed(2)} ${y2.toFixed(2)}`;
  }
  return d;
}

/** Solid filled circle as a path. */
export function circlePath(cx: number, cy: number, r: number): string {
  const [sx, sy] = pt(cx, cy, r, 0);
  return `M ${sx.toFixed(2)} ${sy.toFixed(2)}${bezierArc(cx, cy, r, 0, TAU)} Z`;
}

/** Filled donut sector (outer arc out, inner arc back). */
export function ringSector(
  cx: number,
  cy: number,
  rOuter: number,
  rInner: number,
  t1: number,
  t2: number,
): string {
  const [x1, y1] = pt(cx, cy, rOuter, t1);
  const [x2, y2] = pt(cx, cy, rOuter, t2);
  const [x3, y3] = pt(cx, cy, rInner, t2);
  return (
    `M ${x1.toFixed(2)} ${y1.toFixed(2)}` +
    bezierArc(cx, cy, rOuter, t1, t2) +
    ` L ${x3.toFixed(2)} ${y3.toFixed(2)}` +
    bezierArc(cx, cy, rInner, t2, t1) +
    " Z"
  );
}

/** Four segmented radar arcs with gaps on the diagonals. */
export function radarSegments(
  cx: number,
  cy: number,
  r: number,
  width: number,
  gapAngle: number,
): string[] {
  const segs: string[] = [];
  const half = gapAngle / 2;
  for (let k = 0; k < 4; k += 1) {
    const center = Math.PI / 4 + (Math.PI / 2) * k;
    segs.push(ringSector(cx, cy, r + width / 2, r - width / 2, center + half, center + Math.PI / 2 - half));
  }
  return segs;
}

const OCT = "156,16 356,16 496,156 496,356 356,496 156,496 16,356 16,156";
const OCT_PLATE = "161,34 351,34 478,161 478,351 351,478 161,478 34,351 34,161";
const OCT_INNER = "175,62 337,62 450,175 450,337 337,450 175,450 62,337 62,175";
const DELTA = "256,109 383,329 129,329";

/** Polygon points attribute converted to a closed path. */
export function polyPath(points: string): string {
  const nums = points.trim().split(/[\s,]+/).map(Number);
  let d = `M ${nums[0]} ${nums[1]}`;
  for (let i = 2; i < nums.length; i += 2) {
    d += ` L ${nums[i]} ${nums[i + 1]}`;
  }
  return `${d} Z`;
}

const GAP_ANGLE = (10.45 * Math.PI) / 180;
const SAT_T = (315 * Math.PI) / 180;

interface EmblemSvgProps {
  size: number;
  /** Faint concentric orbital rings around the shield. */
  orbitals?: boolean;
}

/**
 * The PockDev emblem as satori-safe filled paths. Identical geometry to
 * public/brand/pockdev-mark.svg.
 */
export function EmblemSvg({ size, orbitals = false }: EmblemSvgProps) {
  const [satX, satY] = pt(256, 256, 170, SAT_T);
  return (
    <svg width={size} height={size} viewBox="0 0 512 512" style={{ display: "flex" }}>
      {orbitals
        ? [
            <path key="o1" d={ringSector(256, 256, 232, 229, 0, TAU)} fill={PALETTE.moss} />,
            <path key="o2" d={ringSector(256, 256, 258, 255.5, 0, TAU)} fill={PALETTE.moss} />,
            <path key="o3" d={circlePath(256, 256 - 232, 12)} fill={PALETTE.ash} />,
            <path
              key="o4"
              d={circlePath(256 + 258 * Math.cos(SAT_T), 256 + 258 * Math.sin(SAT_T), 8)}
              fill={PALETTE.cyan}
            />,
          ]
        : []}
      <path d={polyPath(OCT)} fill={PALETTE.citron} />
      <path d={polyPath(OCT_PLATE)} fill={PALETTE.silt} />
      <path d={polyPath(OCT_INNER)} fill={PALETTE.olive} />
      {radarSegments(256, 256, 170, 14, GAP_ANGLE).map((d, i) => (
        <path key={i} d={d} fill={PALETTE.citron} />
      ))}
      <path d={circlePath(satX, satY, 22)} fill={PALETTE.citron} />
      <path d={polyPath(DELTA)} fill={PALETTE.terracotta} />
      <path d={circlePath(256, 256, 46)} fill={PALETTE.silt} />
      <path d={circlePath(256, 256, 36)} fill={PALETTE.cyan} />
    </svg>
  );
}

/** Small chamfered prism bullet used by the reply indicators. */
export function PrismBullet({ color }: { color: string }) {
  return (
    <svg width={22} height={30} viewBox="0 0 22 30" style={{ display: "flex" }}>
      <path d="M 6 0 L 22 0 L 16 30 L 0 30 Z" fill={color} />
    </svg>
  );
}

/** The 1200x630 WhatsApp and X preview card. */
export function SocialCard() {
  return (
    <div
      style={{
        width: 1200,
        height: 630,
        display: "flex",
        flexDirection: "column",
        backgroundColor: PALETTE.silt,
        padding: "50px 64px 44px",
      }}
    >
      <div style={{ display: "flex", flexDirection: "row", alignItems: "center", gap: 48, flex: 1 }}>
        <div style={{ display: "flex", width: 380, height: 380, alignItems: "center", justifyContent: "center" }}>
          <EmblemSvg size={380} orbitals />
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 26, flex: 1 }}>
          <div style={{ display: "flex", flexDirection: "row", alignItems: "center", gap: 14 }}>
            <PrismBullet color={PALETTE.citron} />
            <span
              style={{
                display: "flex",
                color: PALETTE.citron,
                fontFamily: "IBM Plex Mono",
                fontWeight: 500,
                fontSize: 21,
                letterSpacing: 5,
              }}
            >
              WEBHOOK SHIELD &amp; POCKET SRE
            </span>
          </div>
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              color: PALETTE.chalk,
              fontFamily: "Syne",
              fontWeight: 800,
              fontSize: 47,
              lineHeight: 1.12,
            }}
          >
            <span style={{ display: "flex" }}>Catch every webhook.</span>
            <span style={{ display: "flex" }}>Fix crashes on WhatsApp.</span>
          </div>
          <div style={{ display: "flex", flexDirection: "row", alignItems: "center", gap: 12 }}>
            <PrismBullet color={PALETTE.cyan} />
            <span
              style={{
                display: "flex",
                color: PALETTE.ash,
                fontFamily: "IBM Plex Mono",
                fontWeight: 500,
                fontSize: 22,
                letterSpacing: 2,
              }}
            >
              pockdev.vercel.app
            </span>
          </div>
        </div>
      </div>

      <div style={{ display: "flex", width: "100%", height: 3, backgroundColor: PALETTE.moss }} />

      <div style={{ display: "flex", flexDirection: "row", alignItems: "center", gap: 22, paddingTop: 30 }}>
        <div style={{ display: "flex", flexDirection: "row", alignItems: "center", gap: 12 }}>
          <PrismBullet color={PALETTE.citron} />
          <span
            style={{
              display: "flex",
              color: PALETTE.chalk,
              fontFamily: "IBM Plex Mono",
              fontWeight: 600,
              fontSize: 25,
            }}
          >
            1: Draft Fix PR
          </span>
        </div>
        <span style={{ display: "flex", color: PALETTE.moss, fontFamily: "IBM Plex Mono", fontSize: 28 }}>/</span>
        <div style={{ display: "flex", flexDirection: "row", alignItems: "center", gap: 12 }}>
          <PrismBullet color={PALETTE.cyan} />
          <span
            style={{
              display: "flex",
              color: PALETTE.chalk,
              fontFamily: "IBM Plex Mono",
              fontWeight: 600,
              fontSize: 25,
            }}
          >
            2: Replay Held
          </span>
        </div>
        <span style={{ display: "flex", color: PALETTE.moss, fontFamily: "IBM Plex Mono", fontSize: 28 }}>/</span>
        <div style={{ display: "flex", flexDirection: "row", alignItems: "center", gap: 12 }}>
          <PrismBullet color={PALETTE.terracotta} />
          <span
            style={{
              display: "flex",
              color: PALETTE.chalk,
              fontFamily: "IBM Plex Mono",
              fontWeight: 600,
              fontSize: 25,
            }}
          >
            3: Mute 1h
          </span>
        </div>
      </div>
    </div>
  );
}

let cachedFonts: { name: string; data: Buffer; weight: 400 | 500 | 600 | 700 | 800; style: "normal" }[] | null = null;

/** Brand fonts for ImageResponse: Syne plus IBM Plex Mono. */
export function loadBrandFonts() {
  if (cachedFonts) {
    return cachedFonts;
  }
  const dir = path.join(process.cwd(), "src", "og-fonts");
  cachedFonts = [
    {
      name: "Syne",
      data: fs.readFileSync(path.join(dir, "Syne-Bold.ttf")),
      weight: 700,
      style: "normal",
    },
    {
      name: "Syne",
      data: fs.readFileSync(path.join(dir, "Syne-ExtraBold.ttf")),
      weight: 800,
      style: "normal",
    },
    {
      name: "IBM Plex Mono",
      data: fs.readFileSync(path.join(dir, "ibm-plex-mono-latin-400-normal.woff")),
      weight: 400,
      style: "normal",
    },
    {
      name: "IBM Plex Mono",
      data: fs.readFileSync(path.join(dir, "ibm-plex-mono-latin-500-normal.woff")),
      weight: 500,
      style: "normal",
    },
    {
      name: "IBM Plex Mono",
      data: fs.readFileSync(path.join(dir, "ibm-plex-mono-latin-600-normal.woff")),
      weight: 600,
      style: "normal",
    },
  ];
  return cachedFonts;
}
