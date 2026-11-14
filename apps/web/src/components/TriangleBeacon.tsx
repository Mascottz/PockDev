"use client";

import type { IncidentStatus } from "@/lib/types";

interface TriangleBeaconProps {
  status: IncidentStatus;
  size?: number;
  pulse?: boolean;
}

const COLORS: Record<IncidentStatus, { stroke: string; fill: string; core: string }> = {
  open: {
    stroke: "#E05A33",
    fill: "rgba(224, 90, 51, 0.16)",
    core: "#E05A33",
  },
  fixing: {
    stroke: "#2AD4C5",
    fill: "rgba(42, 212, 197, 0.16)",
    core: "#2AD4C5",
  },
  muted: {
    stroke: "#A5A196",
    fill: "rgba(165, 161, 150, 0.12)",
    core: "#A5A196",
  },
  resolved: {
    stroke: "#C9F23D",
    fill: "rgba(201, 242, 61, 0.14)",
    core: "#C9F23D",
  },
};

export function TriangleBeacon({
  status,
  size = 44,
  pulse = true,
}: TriangleBeaconProps) {
  const c = COLORS[status];

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      className="overflow-visible shrink-0"
      aria-hidden="true"
    >
      {pulse && status === "open" ? (
        <polygon
          points="32,8 56,50 8,50"
          fill="none"
          stroke={c.stroke}
          strokeWidth="1.5"
          className="beacon-shockwave"
        />
      ) : null}

      <g className={status === "fixing" ? "orbital-cw-med" : ""}>
        <polygon
          points="32,9 54,48 10,48"
          fill={c.fill}
          stroke={c.stroke}
          strokeWidth="2"
        />
      </g>

      {status === "resolved" ? (
        <polyline
          points="24,35 30,41 42,27"
          fill="none"
          stroke={c.core}
          strokeWidth="2.4"
          strokeLinecap="square"
        />
      ) : (
        <>
          <line
            x1="32"
            y1="23"
            x2="32"
            y2="36"
            stroke={c.core}
            strokeWidth="2.4"
          />
          <polygon points="32,40 34.6,44.5 29.4,44.5" fill={c.core} />
        </>
      )}
    </svg>
  );
}
