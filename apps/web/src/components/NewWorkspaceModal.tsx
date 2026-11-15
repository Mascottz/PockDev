"use client";

import { useEffect, useState } from "react";
import type { WorkspaceSummary } from "@/lib/types";
import { GeometricVessel } from "./GeometricVessel";
import { PrismButton } from "./PrismButton";

interface NewWorkspaceModalProps {
  open: boolean;
  onClose: () => void;
  onCreate: (name: string) => Promise<WorkspaceSummary>;
}

export function NewWorkspaceModal({
  open,
  onClose,
  onCreate,
}: NewWorkspaceModalProps) {
  const [name, setName] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  const handleClose = () => {
    setName("");
    setError("");
    onClose();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) {
      setError("Please enter a workspace name.");
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      await onCreate(trimmed);
      setName("");
      onClose();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Could not create workspace."
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-silt/85 p-4 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-label="Create a new PockDev workspace"
    >
      <div className="w-full max-w-lg">
        <GeometricVessel
          shape="diamond-cut"
          tone="cyan"
          innerClassName="p-6 sm:p-7"
        >
          <div className="flex items-start justify-between gap-4 border-b border-moss pb-4">
            <div>
              <span className="font-mono text-[11px] uppercase tracking-[0.2em] text-cyan">
                New Workspace
              </span>
              <h2 className="mt-1 font-display text-2xl font-extrabold text-chalk">
                Create a separate orbit
              </h2>
            </div>
            <PrismButton variant="olive" size="sm" onClick={handleClose}>
              Close
            </PrismButton>
          </div>

          <form onSubmit={handleSubmit} className="mt-5 space-y-4">
            <p className="text-sm leading-relaxed text-ash">
              Keep each team or client isolated with its own shielded apps,
              webhook stream, and active issues.
            </p>

            <label className="block">
              <span className="font-mono text-[11px] uppercase tracking-wider text-ash">
                Workspace name *
              </span>
              <input
                type="text"
                required
                autoFocus
                placeholder="Kora Payments Team"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="mt-1.5 w-full clip-octagon-sm bg-silt px-4 py-2.5 text-sm text-chalk border border-moss focus:border-cyan focus:outline-none"
              />
            </label>

            {error ? (
              <div className="clip-prism-sm bg-terracotta/20 border-l-2 border-terracotta px-4 py-2 font-mono text-xs text-terracotta">
                {error}
              </div>
            ) : null}

            <div className="flex flex-wrap items-center justify-end gap-3 pt-2">
              <PrismButton
                type="button"
                variant="olive"
                size="md"
                onClick={handleClose}
              >
                Cancel
              </PrismButton>
              <PrismButton
                type="submit"
                variant="cyan"
                size="md"
                disabled={submitting}
              >
                {submitting ? "Creating..." : "Create Workspace"}
              </PrismButton>
            </div>
          </form>
        </GeometricVessel>
      </div>
    </div>
  );
}
