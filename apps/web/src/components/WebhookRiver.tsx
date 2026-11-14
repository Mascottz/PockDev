"use client";

import { useState } from "react";
import type { Delivery, DeliveryStatus, Project } from "@/lib/types";
import { GeometricVessel } from "./GeometricVessel";
import { PrismButton } from "./PrismButton";

interface WebhookRiverProps {
  deliveries: Delivery[];
  projects: Project[];
  statusFilter: string;
  onStatusFilterChange: (status: string) => void;
  replayingId: string | null;
  onReplay: (deliveryId: string) => void;
  onOpenIncident: (incidentId: string) => void;
}

const STATUS_TABS: Array<{ value: string; label: string }> = [
  { value: "", label: "All events" },
  { value: "paused", label: "Safely held" },
  { value: "retrying", label: "Retrying" },
  { value: "delivered", label: "Delivered" },
  { value: "failed", label: "Failed" },
];

function prettyPayload(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

function statusMeta(status: DeliveryStatus): {
  label: string;
  tone: "citron" | "terracotta" | "cyan" | "moss";
  badgeClass: string;
} {
  switch (status) {
    case "delivered":
      return {
        label: "Delivered",
        tone: "moss",
        badgeClass: "bg-citron/15 text-citron",
      };
    case "paused":
      return {
        label: "Safely held",
        tone: "terracotta",
        badgeClass: "bg-terracotta text-silt font-bold",
      };
    case "retrying":
      return {
        label: "Retrying",
        tone: "cyan",
        badgeClass: "bg-cyan text-silt font-bold",
      };
    case "failed":
      return {
        label: "Failed",
        tone: "terracotta",
        badgeClass: "bg-terracotta/25 text-terracotta",
      };
  }
}

function formatTime(iso: string): string {
  if (!iso) return "";
  const parts = iso.split("T");
  if (parts.length < 2) return iso;
  return parts[1].slice(0, 8) + " UTC";
}

export function WebhookRiver({
  deliveries,
  projects,
  statusFilter,
  onStatusFilterChange,
  replayingId,
  onReplay,
  onOpenIncident,
}: WebhookRiverProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null);

  return (
    <section className="mt-10">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <span className="font-mono text-[11px] uppercase tracking-[0.2em] text-cyan">
            Live Webhook River
          </span>
          <h2 className="mt-1 font-display text-2xl font-bold text-chalk">
            Caught deliveries and schema drift watch
          </h2>
        </div>

        {/* Diagonal prism filter tabs */}
        <div className="flex flex-wrap items-center gap-2">
          {STATUS_TABS.map((tab) => {
            const active = statusFilter === tab.value;
            return (
              <button
                key={tab.value || "all"}
                type="button"
                onClick={() => onStatusFilterChange(tab.value)}
                className={`clip-prism-sm px-4 py-1.5 font-mono text-xs uppercase tracking-wider transition-colors ${
                  active
                    ? "bg-citron text-silt font-bold"
                    : "bg-olive text-ash hover:bg-basalt hover:text-chalk"
                }`}
              >
                {tab.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Stream of hexagonal/octagonal vessels linked by vertical SVG conduit */}
      <div className="relative mt-6 pl-7 sm:pl-10">
        {/* Animated vertical SVG flow spine */}
        <svg
          className="pointer-events-none absolute left-1.5 top-2 h-[calc(100%-16px)] w-5 sm:left-3"
          preserveAspectRatio="none"
          aria-hidden="true"
        >
          <line
            x1="10"
            y1="0"
            x2="10"
            y2="100%"
            stroke="#2C3527"
            strokeWidth="2"
          />
          <line
            x1="10"
            y1="0"
            x2="10"
            y2="100%"
            stroke="#2AD4C5"
            strokeOpacity="0.7"
            strokeWidth="2"
            className="flow-conduit-calm"
          />
        </svg>

        {deliveries.length === 0 ? (
          <GeometricVessel
            shape="octagon-sm"
            tone="moss"
            innerClassName="p-8 text-center"
          >
            <p className="font-display text-base font-bold text-chalk">
              No webhook events match this filter
            </p>
            <p className="mt-1 text-sm text-ash">
              Switch back to All events or send a test webhook to your Shield URL.
            </p>
          </GeometricVessel>
        ) : (
          <div className="space-y-4">
            {deliveries.map((delivery) => {
              const proj = projects.find((p) => p.id === delivery.projectId);
              const meta = statusMeta(delivery.status);
              const isExpanded = expandedId === delivery.id;
              const isReplaying = replayingId === delivery.id;

              return (
                <div key={delivery.id} className="relative">
                  {/* Spine node marker (pure SVG circle or triangle) */}
                  <svg
                    width="18"
                    height="18"
                    viewBox="0 0 18 18"
                    className="absolute -left-[27px] top-6 sm:-left-[35px]"
                    aria-hidden="true"
                  >
                    {delivery.status === "paused" ||
                    delivery.status === "failed" ? (
                      <polygon
                        points="9,2 16,15 2,15"
                        fill="#0C0E0B"
                        stroke="#E05A33"
                        strokeWidth="2"
                      />
                    ) : delivery.status === "retrying" ? (
                      <polygon
                        points="9,2 16,9 9,16 2,9"
                        fill="#0C0E0B"
                        stroke="#2AD4C5"
                        strokeWidth="2"
                      />
                    ) : (
                      <circle
                        cx="9"
                        cy="9"
                        r="5.5"
                        fill="#0C0E0B"
                        stroke="#C9F23D"
                        strokeWidth="2"
                      />
                    )}
                  </svg>

                  <GeometricVessel
                    shape="octagon-sm"
                    tone={delivery.schemaDriftNote ? "cyan" : meta.tone}
                    innerClassName="p-4 sm:p-5"
                  >
                    <div className="flex flex-wrap items-start justify-between gap-4">
                      <div className="min-w-0 flex-1 space-y-2.5">
                        {/* Metadata bar */}
                        <div className="flex flex-wrap items-center gap-2.5">
                          <span
                            className={`clip-hex-sm px-3.5 py-0.5 font-mono text-[11px] uppercase tracking-wider ${meta.badgeClass}`}
                          >
                            {meta.label}
                          </span>
                          <span className="clip-prism-sm bg-basalt px-3 py-0.5 font-mono text-[11px] text-chalk">
                            {proj?.name ?? delivery.projectId}
                          </span>
                          <span className="font-mono text-xs text-ash">
                            {delivery.id}
                          </span>
                          <span
                            className={`font-mono text-xs font-semibold ${
                              delivery.httpStatus >= 200 &&
                              delivery.httpStatus < 300
                                ? "text-citron"
                                : "text-terracotta"
                            }`}
                          >
                            HTTP {delivery.httpStatus || 0}
                          </span>
                          <span className="font-mono text-xs text-ash">
                            Attempt {delivery.attemptCount}
                          </span>
                          <span className="font-mono text-[11px] text-ash/80">
                            {formatTime(delivery.createdAt)}
                          </span>
                        </div>

                        {/* Schema drift callout prism */}
                        {delivery.schemaDriftNote ? (
                          <div className="clip-prism-sm bg-cyan/15 border-l-2 border-cyan px-4 py-2 flex items-center gap-2.5">
                            <svg
                              width="14"
                              height="14"
                              viewBox="0 0 16 16"
                              className="shrink-0"
                              aria-hidden="true"
                            >
                              <polygon
                                points="8,1 15,8 8,15 1,8"
                                fill="none"
                                stroke="#2AD4C5"
                                strokeWidth="1.8"
                              />
                            </svg>
                            <p className="font-mono text-xs text-cyan">
                              Schema drift detected: {delivery.schemaDriftNote}
                            </p>
                          </div>
                        ) : null}

                        {/* Payload preview */}
                        <div className="clip-octagon-sm bg-silt/95 px-3.5 py-2.5 border border-moss/50">
                          <div className="flex items-center justify-between gap-2">
                            <span className="font-mono text-[10px] uppercase tracking-widest text-ash">
                              Payload snapshot
                            </span>
                            <button
                              type="button"
                              onClick={() =>
                                setExpandedId(isExpanded ? null : delivery.id)
                              }
                              className="font-mono text-[11px] text-cyan hover:text-citron"
                            >
                              {isExpanded ? "Collapse JSON" : "Inspect JSON"}
                            </button>
                          </div>
                          {isExpanded ? (
                            <pre className="mt-2 overflow-x-auto font-mono text-xs leading-relaxed text-chalk">
                              <code>{prettyPayload(delivery.payloadPreview)}</code>
                            </pre>
                          ) : (
                            <p className="mt-1 truncate font-mono text-xs text-ash">
                              {delivery.payloadPreview}
                            </p>
                          )}
                        </div>
                      </div>

                      {/* Right actions */}
                      <div className="flex flex-wrap items-center gap-2.5 sm:flex-col sm:items-end">
                        <PrismButton
                          variant={
                            delivery.status === "delivered" ? "olive" : "citron"
                          }
                          size="sm"
                          disabled={isReplaying}
                          onClick={() => onReplay(delivery.id)}
                        >
                          {isReplaying ? "Sending..." : "Send again"}
                        </PrismButton>

                        {delivery.linkedIncidentId ? (
                          <PrismButton
                            variant="terracotta"
                            size="sm"
                            onClick={() =>
                              onOpenIncident(delivery.linkedIncidentId)
                            }
                          >
                            <svg
                              width="10"
                              height="10"
                              viewBox="0 0 12 12"
                              aria-hidden="true"
                            >
                              <polygon
                                points="6,1 11,10 1,10"
                                fill="currentColor"
                              />
                            </svg>
                            Open issue
                          </PrismButton>
                        ) : null}
                      </div>
                    </div>
                  </GeometricVessel>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </section>
  );
}
