"use client";

import { useId } from "react";

interface GitHubMarkProps {
  size?: number;
  className?: string;
}

export function GitHubMark({ size = 16, className = "" }: GitHubMarkProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="currentColor"
      className={className}
      aria-hidden="true"
    >
      <path d="M12 1.5C6.2 1.5 1.5 6.2 1.5 12c0 4.64 3.01 8.58 7.18 9.97.53.1.72-.23.72-.51 0-.25-.01-1.08-.01-1.96-2.92.63-3.54-1.24-3.54-1.24-.48-1.21-1.17-1.53-1.17-1.53-.95-.65.07-.64.07-.64 1.06.07 1.61 1.08 1.61 1.08.94 1.61 2.46 1.15 3.06.88.1-.68.37-1.15.67-1.41-2.33-.26-4.78-1.17-4.78-5.19 0-1.15.41-2.08 1.08-2.82-.11-.26-.47-1.33.1-2.78 0 0 .88-.28 2.89 1.08A10.1 10.1 0 0 1 12 6.8c.89 0 1.79.12 2.63.36 2.01-1.36 2.89-1.08 2.89-1.08.57 1.45.21 2.52.1 2.78.67.74 1.08 1.67 1.08 2.82 0 4.03-2.46 4.92-4.8 5.18.38.33.71.97.71 1.95 0 1.41-.01 2.54-.01 2.89 0 .28.19.62.73.51A10.51 10.51 0 0 0 22.5 12C22.5 6.2 17.8 1.5 12 1.5Z" />
    </svg>
  );
}

interface OctagonAvatarProps {
  login: string;
  avatarUrl?: string;
  size?: number;
  tone?: "citron" | "cyan";
}

export function OctagonAvatar({
  login,
  avatarUrl,
  size = 28,
  tone = "citron",
}: OctagonAvatarProps) {
  const rawId = useId();
  const clipId = `oct-avatar-${rawId.replace(/[^a-zA-Z0-9_-]/g, "")}`;
  const stroke = tone === "cyan" ? "#2AD4C5" : "#C9F23D";
  const initials = (login.replace(/^@/, "").slice(0, 2) || "PD").toUpperCase();

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 36 36"
      className="shrink-0"
      aria-hidden="true"
    >
      <defs>
        <clipPath id={clipId}>
          <polygon points="10,2 26,2 34,10 34,26 26,34 10,34 2,26 2,10" />
        </clipPath>
      </defs>
      <polygon
        points="10,2 26,2 34,10 34,26 26,34 10,34 2,26 2,10"
        fill="#151913"
      />
      {avatarUrl ? (
        <image
          href={avatarUrl}
          x="2"
          y="2"
          width="32"
          height="32"
          preserveAspectRatio="xMidYMid slice"
          clipPath={`url(#${clipId})`}
        />
      ) : (
        <text
          x="18"
          y="22"
          textAnchor="middle"
          fill={stroke}
          fontSize="12"
          fontWeight="700"
          fontFamily="monospace"
        >
          {initials}
        </text>
      )}
      <polygon
        points="10,2 26,2 34,10 34,26 26,34 10,34 2,26 2,10"
        fill="none"
        stroke={stroke}
        strokeWidth="1.6"
      />
    </svg>
  );
}
