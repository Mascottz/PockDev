"use client";

export type OrbitState = "healthy" | "incident" | "fixing" | "muted";

interface OrbitalNodeProps {
  state: OrbitState;
  size?: number;
  label?: string;
  sublabel?: string;
  active?: boolean;
}

const STATE_PALETTE: Record<
  OrbitState,
  { ring: string; satellite: string; core: string; speedClass: string }
> = {
  healthy: {
    ring: "#C9F23D",
    satellite: "#C9F23D",
    core: "#1E241B",
    speedClass: "orbital-cw-slow",
  },
  incident: {
    ring: "#E05A33",
    satellite: "#E05A33",
    core: "#2E150E",
    speedClass: "orbital-cw-fast",
  },
  fixing: {
    ring: "#2AD4C5",
    satellite: "#C9F23D",
    core: "#0F2624",
    speedClass: "orbital-cw-med",
  },
  muted: {
    ring: "#A5A196",
    satellite: "#A5A196",
    core: "#1C1E19",
    speedClass: "orbital-cw-slow",
  },
};

export function OrbitalNode({
  state,
  size = 96,
  label,
  sublabel,
  active = false,
}: OrbitalNodeProps) {
  const palette = STATE_PALETTE[state];

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 120 120"
      className="overflow-visible select-none"
      aria-hidden={!label}
    >
      {/* Outer breathing halo */}
      <circle
        cx="60"
        cy="60"
        r="52"
        fill="none"
        stroke={palette.ring}
        strokeOpacity={active ? 0.55 : 0.24}
        strokeWidth={active ? 1.5 : 1}
        strokeDasharray="4 6"
        className="orbital-breath"
      />

      {/* Mid counter-rotating guide ring */}
      <g className="orbital-ccw">
        <circle
          cx="60"
          cy="60"
          r="41"
          fill="none"
          stroke={palette.ring}
          strokeOpacity={0.42}
          strokeWidth="1.2"
          strokeDasharray="24 14 6 14"
        />
      </g>

      {/* Primary satellite orbit */}
      <g className={palette.speedClass}>
        <circle
          cx="60"
          cy="60"
          r="31"
          fill="none"
          stroke={palette.ring}
          strokeOpacity={0.65}
          strokeWidth="1.3"
        />
        {state === "incident" ? (
          <polygon
            points="60,22 65,31 55,31"
            fill={palette.satellite}
          />
        ) : (
          <circle cx="60" cy="29" r="3.6" fill={palette.satellite} />
        )}
        <circle
          cx="86"
          cy="76"
          r="2.2"
          fill={palette.ring}
          fillOpacity={0.75}
        />
      </g>

      {/* Core circle */}
      <circle
        cx="60"
        cy="60"
        r="21"
        fill={palette.core}
        stroke={palette.ring}
        strokeWidth={active ? 2.2 : 1.5}
      />

      {/* Center geometric emblem */}
      {state === "incident" ? (
        <polygon
          points="60,48 70,66 50,66"
          fill="none"
          stroke="#E05A33"
          strokeWidth="2"
          className="animate-triangle-pulse svg-origin-center"
        />
      ) : state === "fixing" ? (
        <polygon
          points="60,49 69,60 60,71 51,60"
          fill="none"
          stroke="#2AD4C5"
          strokeWidth="2"
        />
      ) : (
        <circle
          cx="60"
          cy="60"
          r="6"
          fill={palette.ring}
          fillOpacity={0.9}
        />
      )}

      {label ? (
        <text
          x="60"
          y="108"
          textAnchor="middle"
          fill="#F3EFE6"
          fontSize="11"
          fontWeight="600"
        >
          {label}
        </text>
      ) : null}
      {sublabel ? (
        <text
          x="60"
          y="120"
          textAnchor="middle"
          fill="#A5A196"
          fontSize="9"
        >
          {sublabel}
        </text>
      ) : null}
    </svg>
  );
}
