interface BrandMarkProps {
  /** Rendered width and height in pixels. */
  size?: number;
  /** Spin the segmented orbital ring and satellite node smoothly. */
  animated?: boolean;
  className?: string;
}

/**
 * The PockDev geometric emblem: a chamfered octagon shield with a segmented
 * citron orbital ring, a terracotta delta beacon, and a lagoon cyan core.
 * Same geometry as public/brand/pockdev-mark.svg so tabs, pages, and
 * downloads stay pixel consistent.
 */
export function BrandMark({ size = 40, animated = false, className = "" }: BrandMarkProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 512 512"
      className={`overflow-visible ${className}`}
      role="img"
      aria-label="PockDev mark"
    >
      <polygon
        points="156,16 356,16 496,156 496,356 356,496 156,496 16,356 16,156"
        fill="#C9F23D"
      />
      <polygon
        points="161,34 351,34 478,161 478,351 351,478 161,478 34,351 34,161"
        fill="#0C0E0B"
      />
      <polygon
        points="175,62 337,62 450,175 450,337 337,450 175,450 62,337 62,175"
        fill="#151913"
        stroke="#2C3527"
        strokeWidth="3"
      />
      <g className={animated ? "orbital-cw-slow" : undefined}>
        {/* Invisible bounds keep the CSS rotation origin dead center. */}
        <circle cx="256" cy="256" r="210" fill="none" stroke="none" />
        <circle
          cx="256"
          cy="256"
          r="170"
          fill="none"
          stroke="#C9F23D"
          strokeWidth="14"
          strokeDasharray="236 31"
          strokeDashoffset="118"
        />
        <circle cx="376.2" cy="135.8" r="22" fill="#C9F23D" />
      </g>
      <polygon points="256,109 383,329 129,329" fill="#E05A33" />
      <circle cx="256" cy="256" r="46" fill="#0C0E0B" />
      <circle cx="256" cy="256" r="36" fill="#2AD4C5" />
    </svg>
  );
}
