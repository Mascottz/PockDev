"use client";

import { useEffect, useState } from "react";
import type { Incident, IncidentAction, Project } from "@/lib/types";
import { GeometricVessel } from "./GeometricVessel";
import { PrismButton } from "./PrismButton";
import { TriangleBeacon } from "./TriangleBeacon";

interface WhatsAppTurn {
  from: "user" | "pockdev";
  text: string;
}

interface IncidentDrawerProps {
  incident: Incident | null;
  project?: Project;
  busyAction: IncidentAction | null;
  actionFeedback: string;
  onClose: () => void;
  onAction: (incidentId: string, action: IncidentAction) => Promise<void>;
}

export function IncidentDrawer({
  incident,
  project,
  busyAction,
  actionFeedback,
  onClose,
  onAction,
}: IncidentDrawerProps) {
  const [thread, setThread] = useState<WhatsAppTurn[]>([]);

  useEffect(() => {
    setThread([]);
  }, [incident?.id]);

  useEffect(() => {
    if (!incident) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [incident, onClose]);

  if (!incident) return null;

  const tone =
    incident.status === "open"
      ? "terracotta"
      : incident.status === "fixing"
        ? "cyan"
        : incident.status === "resolved"
          ? "citron"
          : "moss";

  const triggerWithWhatsApp = async (
    action: IncidentAction,
    replyDigit?: "1" | "2" | "3"
  ) => {
    if (replyDigit) {
      setThread((prev) => [...prev, { from: "user", text: replyDigit }]);
    }
    await onAction(incident.id, action);

    const botReply =
      action === "draft_pr"
        ? "Got it. Drafted a fix PR on GitHub with the safe field check. Merge when ready, then reply 2 to send held events."
        : action === "replay_all"
          ? "Sent all safely held events back to your target endpoint. Everything delivered cleanly."
          : action === "mute"
            ? "Muted alerts for this issue for 1 hour. Incoming webhooks will still be safely held."
            : "Marked this issue resolved. All quiet now.";

    setThread((prev) => [...prev, { from: "pockdev", text: botReply }]);
  };

  const heldPhrase =
    incident.pausedCount === 1
      ? "1 event is safely held."
      : `${incident.pausedCount} events are safely held.`;

  return (
    <div
      className="fixed inset-0 z-50 flex justify-end bg-silt/80 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-label="Incident details and WhatsApp control"
    >
      {/* Clickable dim backdrop */}
      <button
        type="button"
        aria-label="Close drawer"
        onClick={onClose}
        className="flex-1 cursor-default"
      />

      {/* Polygon Drawer Panel */}
      <div className="h-full w-full max-w-2xl overflow-y-auto p-3 sm:p-5">
        <GeometricVessel
          shape="diamond-cut"
          tone={tone}
          className="min-h-full"
          innerClassName="p-5 sm:p-7 flex flex-col justify-between gap-6"
        >
          <div className="space-y-6">
            {/* Drawer Header */}
            <div className="flex items-start justify-between gap-4 border-b border-moss pb-4">
              <div className="flex items-start gap-3.5">
                <TriangleBeacon status={incident.status} size={48} />
                <div>
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-mono text-xs uppercase tracking-wider text-ash">
                      {project?.name ?? incident.projectId}
                    </span>
                    <span
                      className={`clip-hex-sm px-3 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider ${
                        incident.status === "open"
                          ? "bg-terracotta text-silt"
                          : incident.status === "fixing"
                            ? "bg-cyan text-silt"
                            : incident.status === "resolved"
                              ? "bg-citron text-silt"
                              : "bg-moss text-chalk"
                      }`}
                    >
                      {incident.status === "open"
                        ? "Open issue"
                        : incident.status === "fixing"
                          ? "Fix PR open"
                          : incident.status === "muted"
                            ? "Muted 1h"
                            : "All quiet"}
                    </span>
                  </div>
                  <h2 className="mt-1.5 font-display text-xl font-extrabold leading-snug text-chalk">
                    {incident.title}
                  </h2>
                </div>
              </div>

              <PrismButton variant="olive" size="sm" onClick={onClose}>
                Close
              </PrismButton>
            </div>

            {/* Plain-English diagnosis */}
            <GeometricVessel
              shape="octagon-sm"
              tone="moss"
              innerClassName="p-4 sm:p-5 bg-silt"
            >
              <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-citron">
                Plain-English Diagnosis
              </span>
              <p className="mt-2 text-sm leading-relaxed text-chalk sm:text-base">
                {incident.plainSummary ||
                  "PockDev linked this crash to held webhook deliveries and is ready for your next move."}
              </p>

              {/* File + Commit + Held Count Strip */}
              <div className="mt-4 grid gap-2.5 sm:grid-cols-3 border-t border-moss/60 pt-3.5 font-mono text-xs">
                <div>
                  <span className="block text-[10px] uppercase tracking-wider text-ash">
                    Source line
                  </span>
                  <span className="text-cyan">
                    {incident.filePath || "webhook dead letter"}
                  </span>
                </div>
                <div>
                  <span className="block text-[10px] uppercase tracking-wider text-ash">
                    Recent commit
                  </span>
                  <span className="text-chalk">
                    {incident.commitSha || "pending scan"}
                  </span>
                </div>
                <div>
                  <span className="block text-[10px] uppercase tracking-wider text-ash">
                    Webhook shield
                  </span>
                  <span
                    className={
                      incident.pausedCount > 0
                        ? "text-terracotta font-semibold"
                        : "text-citron"
                    }
                  >
                    {incident.pausedCount > 0
                      ? `${incident.pausedCount} safely held`
                      : "0 held"}
                  </span>
                </div>
              </div>

              {incident.prUrl ? (
                <div className="mt-4 clip-prism-sm bg-cyan/15 border-l-2 border-cyan px-4 py-2.5 flex flex-wrap items-center justify-between gap-2">
                  <span className="font-mono text-xs text-cyan">
                    Fix PR drafted: {incident.prUrl}
                  </span>
                  <a
                    href={incident.prUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="font-mono text-xs font-bold uppercase text-citron hover:underline"
                  >
                    Open PR -&gt;
                  </a>
                </div>
              ) : null}
            </GeometricVessel>

            {/* Live WhatsApp Phone Preview inside Chamfered Octagonal Handset */}
            <GeometricVessel
              shape="octagon"
              tone="cyan"
              innerClassName="p-4 sm:p-5 bg-silt"
            >
              <div className="flex flex-wrap items-center justify-between gap-2 border-b border-moss pb-3">
                <div className="flex items-center gap-2.5">
                  <svg width="18" height="18" viewBox="0 0 20 20" aria-hidden="true">
                    <circle
                      cx="10"
                      cy="10"
                      r="8"
                      fill="none"
                      stroke="#2AD4C5"
                      strokeWidth="1.6"
                    />
                    <circle cx="10" cy="10" r="3" fill="#C9F23D" />
                  </svg>
                  <div>
                    <p className="font-mono text-xs font-semibold uppercase tracking-wider text-chalk">
                      WhatsApp Pocket Preview
                    </p>
                    <p className="font-mono text-[10px] text-ash">
                      To {project?.whatsappPhone || "+234 809 441 2201"}
                    </p>
                  </div>
                </div>
                <span className="clip-hex-sm bg-basalt px-3 py-0.5 font-mono text-[10px] uppercase tracking-wider text-cyan">
                  Interactive reply
                </span>
              </div>

              {/* Message Thread */}
              <div className="mt-4 space-y-3 text-xs leading-relaxed">
                <div className="clip-bevel-chat bg-olive p-3.5 text-chalk">
                  <p>{incident.plainSummary}</p>
                  <p className="mt-2 font-medium text-citron">
                    {heldPhrase} Reply 1 to draft a fix PR, 2 to send paused
                    events again, or 3 to mute for 1 hour.
                  </p>
                </div>

                {thread.map((turn, idx) =>
                  turn.from === "user" ? (
                    <div
                      key={`turn-${idx}`}
                      className="ml-auto max-w-[160px] clip-prism-sm bg-citron px-4 py-1.5 text-right font-mono text-xs font-bold text-silt"
                    >
                      Reply {turn.text}
                    </div>
                  ) : (
                    <div
                      key={`turn-${idx}`}
                      className="clip-bevel-chat bg-basalt p-3.5 text-cyan"
                    >
                      {turn.text}
                    </div>
                  )
                )}
              </div>

              {/* Tap-to-reply 1 / 2 / 3 buttons */}
              <div className="mt-4 flex flex-wrap items-center gap-2 border-t border-moss/60 pt-3.5">
                <span className="font-mono text-[10px] uppercase tracking-wider text-ash mr-1">
                  Quick reply:
                </span>
                <PrismButton
                  variant="cyan"
                  size="sm"
                  disabled={busyAction !== null}
                  onClick={() => triggerWithWhatsApp("draft_pr", "1")}
                >
                  Reply 1: Fix PR
                </PrismButton>
                <PrismButton
                  variant="citron"
                  size="sm"
                  disabled={busyAction !== null}
                  onClick={() => triggerWithWhatsApp("replay_all", "2")}
                >
                  Reply 2: Send held
                </PrismButton>
                <PrismButton
                  variant="olive"
                  size="sm"
                  disabled={busyAction !== null}
                  onClick={() => triggerWithWhatsApp("mute", "3")}
                >
                  Reply 3: Mute 1h
                </PrismButton>
              </div>
            </GeometricVessel>

            {actionFeedback ? (
              <div className="clip-prism-sm bg-citron/15 border-l-2 border-citron px-4 py-2 font-mono text-xs text-citron">
                {actionFeedback}
              </div>
            ) : null}
          </div>

          {/* Direct Action Triggers Footer */}
          <div className="border-t border-moss pt-5">
            <span className="block font-mono text-[10px] uppercase tracking-[0.2em] text-ash mb-3">
              Direct Deck Triggers
            </span>
            <div className="flex flex-wrap items-center gap-2.5">
              <PrismButton
                variant="cyan"
                size="md"
                disabled={busyAction !== null}
                onClick={() => triggerWithWhatsApp("draft_pr")}
              >
                {busyAction === "draft_pr" ? "Drafting PR..." : "Draft fix PR"}
              </PrismButton>

              <PrismButton
                variant="citron"
                size="md"
                disabled={busyAction !== null}
                onClick={() => triggerWithWhatsApp("replay_all")}
              >
                {busyAction === "replay_all"
                  ? "Sending..."
                  : "Send paused events"}
              </PrismButton>

              <PrismButton
                variant="chalk"
                size="md"
                disabled={busyAction !== null}
                onClick={() => triggerWithWhatsApp("resolve")}
              >
                {busyAction === "resolve" ? "Resolving..." : "Resolve"}
              </PrismButton>

              <PrismButton
                variant="olive"
                size="md"
                disabled={busyAction !== null}
                onClick={() => triggerWithWhatsApp("mute")}
              >
                {busyAction === "mute" ? "Muting..." : "Mute 1h"}
              </PrismButton>
            </div>
          </div>
        </GeometricVessel>
      </div>
    </div>
  );
}
