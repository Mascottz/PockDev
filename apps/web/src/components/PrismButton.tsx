"use client";

import type { ButtonHTMLAttributes, ReactNode } from "react";

export type PrismVariant = "citron" | "terracotta" | "cyan" | "olive" | "chalk";

interface PrismButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: PrismVariant;
  shape?: "prism" | "hex";
  size?: "sm" | "md" | "lg";
  children: ReactNode;
}

const OUTER_STYLES: Record<PrismVariant, string> = {
  citron: "bg-citron hover:bg-chalk",
  terracotta: "bg-terracotta hover:bg-chalk",
  cyan: "bg-cyan hover:bg-chalk",
  olive: "bg-moss hover:bg-citron/80",
  chalk: "bg-chalk/70 hover:bg-citron",
};

const INNER_STYLES: Record<PrismVariant, string> = {
  citron: "bg-citron text-silt hover:bg-chalk hover:text-silt",
  terracotta: "bg-terracotta text-silt hover:bg-terracotta/90 hover:text-silt",
  cyan: "bg-cyan text-silt hover:bg-cyan/90 hover:text-silt",
  olive: "bg-olive text-chalk hover:bg-basalt",
  chalk: "bg-silt text-chalk hover:bg-basalt",
};

const SIZE_STYLES: Record<"sm" | "md" | "lg", string> = {
  sm: "px-4 py-1.5 text-xs tracking-wide",
  md: "px-6 py-2.5 text-sm tracking-wide",
  lg: "px-8 py-3.5 text-sm tracking-wider",
};

export function PrismButton({
  variant = "citron",
  shape = "prism",
  size = "md",
  className = "",
  disabled,
  children,
  ...rest
}: PrismButtonProps) {
  const clip = shape === "hex" ? "clip-hex-sm" : size === "sm" ? "clip-prism-sm" : "clip-prism";

  return (
    <button
      {...rest}
      disabled={disabled}
      className={`group inline-block ${clip} ${OUTER_STYLES[variant]} p-[1px] transition-transform duration-150 active:scale-[0.98] disabled:opacity-45 disabled:pointer-events-none ${className}`}
    >
      <span
        className={`flex items-center justify-center gap-2 ${clip} ${INNER_STYLES[variant]} ${SIZE_STYLES[size]} font-semibold uppercase transition-colors duration-150`}
      >
        {children}
      </span>
    </button>
  );
}
