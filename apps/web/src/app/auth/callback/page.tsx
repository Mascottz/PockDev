"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { GeometricVessel } from "@/components/GeometricVessel";
import { PrismButton } from "@/components/PrismButton";
import { TriangleBeacon } from "@/components/TriangleBeacon";
import {
  loadEngineSettings,
  saveCachedAuthUser,
  saveEngineSettings,
} from "@/lib/engine-client";

type CallbackState = "working" | "ready" | "error";

function describeAuthError(code: string): string {
  const clean = code.trim();
  if (!clean) {
    return "GitHub sign-in did not return a session token. Head back to the Control Deck and try again.";
  }
  switch (clean.toLowerCase()) {
    case "access_denied":
      return "GitHub sign-in was cancelled before finishing. You can try again anytime from the Control Deck.";
    case "invalid_state":
    case "state_mismatch":
      return "Your sign-in link expired or did not match this browser session. Please start GitHub sign-in again.";
    case "github_not_configured":
      return "GitHub sign-in is not configured on this PockDev engine yet. You can still connect using an API token.";
    default:
      return `Sign-in could not finish (${clean}). Head back to the Control Deck to try again.`;
  }
}

export default function AuthCallbackPage() {
  const router = useRouter();
  const [state, setState] = useState<CallbackState>("working");
  const [message, setMessage] = useState(
    "Linking your GitHub session to the Control Deck..."
  );

  useEffect(() => {
    const rawHash = window.location.hash.replace(/^#/, "");
    const hashParams = new URLSearchParams(rawHash);
    const searchParams = new URLSearchParams(window.location.search);

    const session = (hashParams.get("session") || "").trim();
    const errParam = (
      searchParams.get("error") ||
      hashParams.get("error") ||
      ""
    ).trim();

    if (session) {
      const current = loadEngineSettings();
      saveCachedAuthUser(null);
      saveEngineSettings({
        ...current,
        token: session,
        workspaceId: "",
      });
      window.history.replaceState(null, "", "/auth/callback");
      setState("ready");
      setMessage("Signed in with GitHub. Opening your Control Deck...");
      router.replace("/deck");
      return;
    }

    if (rawHash) {
      window.history.replaceState(null, "", window.location.pathname + window.location.search);
    }
    setState("error");
    setMessage(describeAuthError(errParam));
  }, [router]);

  const tone =
    state === "error" ? "terracotta" : state === "ready" ? "citron" : "cyan";

  return (
    <div className="mx-auto flex min-h-[70vh] max-w-xl items-center justify-center px-5 py-16">
      <GeometricVessel
        shape="octagon"
        tone={tone}
        className="w-full"
        innerClassName="p-7 sm:p-10"
      >
        <div className="flex flex-col items-center text-center">
          {state === "error" ? (
            <TriangleBeacon status="open" size={58} />
          ) : (
            <svg
              width="64"
              height="64"
              viewBox="0 0 64 64"
              className="overflow-visible"
              aria-hidden="true"
            >
              <circle
                cx="32"
                cy="32"
                r="26"
                fill="none"
                stroke={state === "ready" ? "#C9F23D" : "#2AD4C5"}
                strokeWidth="1.5"
                strokeDasharray="10 8"
                className="orbital-cw-med"
              />
              <circle
                cx="32"
                cy="32"
                r="17"
                fill="#0C0E0B"
                stroke={state === "ready" ? "#C9F23D" : "#2AD4C5"}
                strokeWidth="2"
              />
              {state === "ready" ? (
                <polyline
                  points="24,33 30,39 41,26"
                  fill="none"
                  stroke="#C9F23D"
                  strokeWidth="2.4"
                  strokeLinecap="square"
                />
              ) : (
                <circle cx="32" cy="32" r="5" fill="#2AD4C5" />
              )}
            </svg>
          )}

          <span
            className={`mt-5 clip-hex-sm px-3.5 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.2em] ${
              state === "error"
                ? "bg-terracotta text-silt"
                : state === "ready"
                  ? "bg-citron text-silt"
                  : "bg-cyan text-silt"
            }`}
          >
            {state === "error"
              ? "Sign-in paused"
              : state === "ready"
                ? "Session active"
                : "Completing sign-in"}
          </span>

          <h1 className="mt-3 font-display text-2xl font-extrabold text-chalk">
            {state === "error"
              ? "Could not finish GitHub sign-in"
              : state === "ready"
                ? "Welcome to PockDev"
                : "Connecting your GitHub account"}
          </h1>

          <p className="mt-2.5 max-w-md text-sm leading-relaxed text-ash">
            {message}
          </p>

          <div className="mt-7 flex flex-wrap items-center justify-center gap-3">
            <Link href="/deck">
              <PrismButton
                variant={state === "error" ? "citron" : "olive"}
                size="md"
              >
                Back to Control Deck
              </PrismButton>
            </Link>
          </div>
        </div>
      </GeometricVessel>
    </div>
  );
}
