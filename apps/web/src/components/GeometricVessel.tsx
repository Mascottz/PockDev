import type { ReactNode } from "react";

export type VesselShape =
  | "octagon"
  | "octagon-sm"
  | "hex-bar"
  | "hex-sm"
  | "prism"
  | "prism-sm"
  | "diamond-cut"
  | "bevel-chat";

export type VesselTone =
  | "moss"
  | "terracotta"
  | "citron"
  | "cyan"
  | "chalk";

const SHAPE_CLASS: Record<VesselShape, string> = {
  octagon: "clip-octagon",
  "octagon-sm": "clip-octagon-sm",
  "hex-bar": "clip-hex-bar",
  "hex-sm": "clip-hex-sm",
  prism: "clip-prism",
  "prism-sm": "clip-prism-sm",
  "diamond-cut": "clip-diamond-cut",
  "bevel-chat": "clip-bevel-chat",
};

const BORDER_TONE: Record<VesselTone, string> = {
  moss: "bg-moss",
  terracotta: "bg-terracotta/80",
  citron: "bg-citron/75",
  cyan: "bg-cyan/75",
  chalk: "bg-chalk/40",
};

interface GeometricVesselProps {
  shape?: VesselShape;
  tone?: VesselTone;
  className?: string;
  innerClassName?: string;
  children: ReactNode;
}

export function GeometricVessel({
  shape = "octagon",
  tone = "moss",
  className = "",
  innerClassName = "",
  children,
}: GeometricVesselProps) {
  const clip = SHAPE_CLASS[shape];
  return (
    <div className={`${clip} ${BORDER_TONE[tone]} p-[1px] ${className}`}>
      <div className={`${clip} bg-olive h-full w-full ${innerClassName}`}>
        {children}
      </div>
    </div>
  );
}
