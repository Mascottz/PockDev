"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  applyDemoIncidentAction,
  applyDemoReplayDelivery,
  computeOverview,
  createDemoProject,
  createSimulatedDrill,
  DEMO_DELIVERIES,
  DEMO_INCIDENTS,
  DEMO_PROJECTS,
} from "@/lib/demo-data";
import {
  buildGitHubLoginUrl,
  clearAuthSession,
  createEngineClient,
  DEFAULT_ENGINE_URL,
  loadEngineSettings,
  saveCachedAuthUser,
  saveEngineSettings,
} from "@/lib/engine-client";
import type {
  BillingStatus,
  CreateProjectInput,
  Delivery,
  EngineSettings,
  Incident,
  IncidentAction,
  MeResponse,
  Overview,
  Project,
  WorkspaceSummary,
} from "@/lib/types";
import { GeometricVessel } from "./GeometricVessel";
import { GitHubMark, OctagonAvatar } from "./GitHubMark";
import { IncidentDrawer } from "./IncidentDrawer";
import { NewProjectModal } from "./NewProjectModal";
import { NewWorkspaceModal } from "./NewWorkspaceModal";
import { OrbitalConstellation } from "./OrbitalConstellation";
import { OrbitalNode, type OrbitState } from "./OrbitalNode";
import { PrismButton } from "./PrismButton";
import { QuotaBar } from "./QuotaBar";
import { WebhookRiver } from "./WebhookRiver";

type ConnectionMode = "connecting" | "live" | "demo";

function formatUptime(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const mins = Math.floor(seconds / 60);
  if (mins < 60) return `${mins}m ${seconds % 60}s`;
  const hrs = Math.floor(mins / 60);
  return `${hrs}h ${mins % 60}m`;
}

interface OrbitalDialProps {
  label: string;
  value: string;
  subtext: string;
  progress: number;
  color: string;
  alert?: boolean;
}

function OrbitalDial({
  label,
  value,
  subtext,
  progress,
  color,
  alert = false,
}: OrbitalDialProps) {
  const radius = 40;
  const circumference = 2 * Math.PI * radius;
  const clamped = Math.max(0.06, Math.min(1, progress));
  const offset = circumference * (1 - clamped);

  return (
    <div className="flex items-center gap-4 px-2 py-2">
      <svg
        width="92"
        height="92"
        viewBox="0 0 100 100"
        className="shrink-0 overflow-visible"
        aria-hidden="true"
      >
        <circle
          cx="50"
          cy="50"
          r="46"
          fill="none"
          stroke="#2C3527"
          strokeWidth="1"
          strokeDasharray="3 6"
        />
        <circle
          cx="50"
          cy="50"
          r={radius}
          fill="#151913"
          stroke="#1E241B"
          strokeWidth="4.5"
        />
        <circle
          cx="50"
          cy="50"
          r={radius}
          fill="none"
          stroke={color}
          strokeWidth="4.5"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          transform="rotate(-90 50 50)"
          className="transition-all duration-500"
        />
        {alert ? (
          <polygon
            points="50,20 56,31 44,31"
            fill="#E05A33"
            className="animate-triangle-pulse svg-origin-center"
          />
        ) : null}
        <text
          x="50"
          y={alert ? "59" : "55"}
          textAnchor="middle"
          fill="#F3EFE6"
          fontSize="18"
          fontWeight="800"
          fontFamily="monospace"
        >
          {value}
        </text>
      </svg>

      <div className="min-w-0">
        <span className="block font-mono text-[10px] uppercase tracking-[0.2em] text-ash">
          {label}
        </span>
        <p className="mt-1 font-display text-sm font-bold text-chalk">
          {subtext}
        </p>
      </div>
    </div>
  );
}

export function ControlDeck() {
  const [settings, setSettings] = useState<EngineSettings>({
    baseUrl: DEFAULT_ENGINE_URL,
    token: "",
    workspaceId: "",
  });
  const [draftUrl, setDraftUrl] = useState(DEFAULT_ENGINE_URL);
  const [draftToken, setDraftToken] = useState("");
  const [configOpen, setConfigOpen] = useState(false);

  const [mode, setMode] = useState<ConnectionMode>("connecting");
  const [uptimeSeconds, setUptimeSeconds] = useState(0);
  const [githubAuthConfigured, setGithubAuthConfigured] = useState<
    boolean | null
  >(null);
  const [authRequired, setAuthRequired] = useState(false);
  const [me, setMe] = useState<MeResponse | null>(null);
  const [signingOut, setSigningOut] = useState(false);
  const [workspaceMenuOpen, setWorkspaceMenuOpen] = useState(false);
  const [newWorkspaceOpen, setNewWorkspaceOpen] = useState(false);
  const workspaceMenuRef = useRef<HTMLDivElement | null>(null);

  const [connectionNote, setConnectionNote] = useState(
    "Checking PockDev Go engine..."
  );

  const [projects, setProjects] = useState<Project[]>(DEMO_PROJECTS);
  const [incidents, setIncidents] = useState<Incident[]>(DEMO_INCIDENTS);
  const [deliveries, setDeliveries] = useState<Delivery[]>(DEMO_DELIVERIES);
  const [liveOverview, setLiveOverview] = useState<Overview | null>(null);
  const [billing, setBilling] = useState<BillingStatus | null>(null);
  const [upgradingBilling, setUpgradingBilling] = useState(false);
  const [verifyingBilling, setVerifyingBilling] = useState(false);
  const [celebrationToast, setCelebrationToast] = useState("");
  const verifiedRefRef = useRef<string | null>(null);

  const [selectedProjectId, setSelectedProjectId] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [activeIncidentId, setActiveIncidentId] = useState<string | null>(null);
  const [newProjectOpen, setNewProjectOpen] = useState(false);

  const [replayingId, setReplayingId] = useState<string | null>(null);
  const [busyAction, setBusyAction] = useState<IncidentAction | null>(null);
  const [actionFeedback, setActionFeedback] = useState("");
  const [bannerNote, setBannerNote] = useState("");

  const notify = useCallback((msg: string) => {
    setBannerNote(msg);
    window.setTimeout(() => {
      setBannerNote((current) => (current === msg ? "" : current));
    }, 3600);
  }, []);

  useEffect(() => {
    if (!workspaceMenuOpen) return;
    const onPointer = (e: MouseEvent) => {
      if (
        workspaceMenuRef.current &&
        !workspaceMenuRef.current.contains(e.target as Node)
      ) {
        setWorkspaceMenuOpen(false);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setWorkspaceMenuOpen(false);
    };
    window.addEventListener("mousedown", onPointer);
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("mousedown", onPointer);
      window.removeEventListener("keydown", onKey);
    };
  }, [workspaceMenuOpen]);

  const syncFromEngine = useCallback(
    async (cfg: EngineSettings, manual = false): Promise<boolean> => {
      setMode("connecting");
      const client = createEngineClient(cfg);
      try {
        const health = await client.getHealth();
        if (!health.ok) throw new Error("Engine reported unhealthy status");
        setUptimeSeconds(health.uptimeSeconds);
        setGithubAuthConfigured(
          typeof health.githubAuthConfigured === "boolean"
            ? health.githubAuthConfigured
            : null
        );

        if (!cfg.token.trim()) {
          setLiveOverview(null);
          setBilling(null);
          setMe(null);
          saveCachedAuthUser(null);
          setAuthRequired(true);
          setMode("demo");
          setConnectionNote(
            "Engine reachable; sign in with GitHub or explore the interactive demo orbit below"
          );
          return false;
        }

        const [ov, prjs, incs, dlvs, meRes, billingRes] = await Promise.all([
          client.getOverview(),
          client.listProjects(),
          client.listIncidents(),
          client.listDeliveries(),
          client.getMe().catch(() => null),
          client.getBilling().catch(() => null),
        ]);

        setLiveOverview(ov);
        setProjects(prjs ?? []);
        setIncidents(incs ?? []);
        setDeliveries(dlvs ?? []);
        setMe(meRes);
        setBilling(billingRes);
        saveCachedAuthUser(meRes?.user ?? null);
        if (meRes?.workspace?.id && meRes.workspace.id !== cfg.workspaceId) {
          const synced: EngineSettings = {
            ...cfg,
            workspaceId: meRes.workspace.id,
          };
          saveEngineSettings(synced);
          setSettings(synced);
        }
        setAuthRequired(false);
        setMode("live");
        setConnectionNote(
          `Engine live; up ${formatUptime(health.uptimeSeconds)}`
        );
        return true;
      } catch (err) {
        setLiveOverview(null);
        setBilling(null);
        setMe(null);
        saveCachedAuthUser(null);
        setMode("demo");
        const msg = err instanceof Error ? err.message : "connection failed";
        const isUnauthorized =
          msg.toLowerCase().includes("token") ||
          msg.toLowerCase().includes("session") ||
          msg.toLowerCase().includes("unauthorized") ||
          msg.includes("401");
        if (isUnauthorized || !cfg.token.trim()) {
          setAuthRequired(true);
          setConnectionNote(
            isUnauthorized
              ? "Session expired or invalid token; sign in with GitHub or check Advanced settings"
              : "Sign in with GitHub to connect live apps, or explore the demo orbit below"
          );
        } else if (manual || !cfg.baseUrl.includes("localhost")) {
          setAuthRequired(true);
          setConnectionNote(`Could not connect (${msg}); showing demo orbit`);
        } else {
          setAuthRequired(true);
          setConnectionNote(
            "Interactive demo mode; sign in with GitHub or connect a local engine"
          );
        }
        return false;
      }
    },
    []
  );

  useEffect(() => {
    const saved = loadEngineSettings();
    setSettings(saved);
    setDraftUrl(saved.baseUrl);
    setDraftToken(saved.token);

    const params = new URLSearchParams(window.location.search);
    const billingFlag = params.get("billing");
    const paystackRef = (
      params.get("reference") ||
      params.get("trxref") ||
      ""
    ).trim();
    const hasVerifyRedirect =
      Boolean(paystackRef) &&
      (billingFlag === "verify" || params.has("trxref"));

    if (hasVerifyRedirect && verifiedRefRef.current !== paystackRef) {
      verifiedRefRef.current = paystackRef;
      params.delete("billing");
      params.delete("reference");
      params.delete("trxref");
      const remaining = params.toString();
      const cleanedUrl = `${window.location.pathname}${remaining ? `?${remaining}` : ""}${window.location.hash}`;
      window.history.replaceState({}, "", cleanedUrl);

      if (saved.token.trim()) {
        setVerifyingBilling(true);
        const client = createEngineClient(saved);
        void client
          .verifyBilling(paystackRef)
          .then(async (verified) => {
            await syncFromEngine(saved);
            setBilling(verified);
            setCelebrationToast(
              "Workspace upgraded to Pro Orbit: 20 apps and 100,000 monthly events unlocked."
            );
          })
          .catch(async (err) => {
            await syncFromEngine(saved);
            notify(
              err instanceof Error
                ? err.message
                : "Could not verify Paystack payment reference."
            );
          })
          .finally(() => {
            setVerifyingBilling(false);
          });
        return;
      }

      notify("Sign in with GitHub to complete your Paystack verification.");
    }

    void syncFromEngine(saved);
  }, [syncFromEngine, notify]);

  const overview: Overview = useMemo(() => {
    if (mode === "live" && liveOverview) return liveOverview;
    return computeOverview(projects, incidents, deliveries);
  }, [mode, liveOverview, projects, incidents, deliveries]);

  const filteredDeliveries = useMemo(() => {
    return deliveries.filter((d) => {
      if (selectedProjectId && d.projectId !== selectedProjectId) return false;
      if (statusFilter && d.status !== statusFilter) return false;
      return true;
    });
  }, [deliveries, selectedProjectId, statusFilter]);

  const filteredIncidents = useMemo(() => {
    if (!selectedProjectId) return incidents;
    return incidents.filter((i) => i.projectId === selectedProjectId);
  }, [incidents, selectedProjectId]);

  const activeIncident = useMemo(
    () => incidents.find((i) => i.id === activeIncidentId) ?? null,
    [incidents, activeIncidentId]
  );

  const activeIncidentProject = useMemo(
    () =>
      activeIncident
        ? projects.find((p) => p.id === activeIncident.projectId)
        : undefined,
    [activeIncident, projects]
  );

  const selectedProject = useMemo(
    () => projects.find((p) => p.id === selectedProjectId) ?? null,
    [projects, selectedProjectId]
  );

  const copyToClipboard = async (value: string, label: string) => {
    try {
      await navigator.clipboard.writeText(value);
      notify(`Copied ${label} to clipboard.`);
    } catch {
      notify(`Could not copy ${label}.`);
    }
  };

  const handleGitHubSignIn = () => {
    const targetBase = draftUrl.trim() || settings.baseUrl || DEFAULT_ENGINE_URL;
    if (targetBase !== settings.baseUrl) {
      const updated = { ...settings, baseUrl: targetBase };
      saveEngineSettings(updated);
      setSettings(updated);
    }
    window.location.href = buildGitHubLoginUrl(
      targetBase,
      window.location.origin
    );
  };

  const handleSignOut = async () => {
    setSigningOut(true);
    try {
      if (settings.token.trim()) {
        const client = createEngineClient(settings);
        await client.logout().catch(() => null);
      }
    } finally {
      const cleared = clearAuthSession();
      setSettings(cleared);
      setDraftToken("");
      setMe(null);
      setBilling(null);
      setCelebrationToast("");
      setLiveOverview(null);
      setProjects(DEMO_PROJECTS);
      setIncidents(DEMO_INCIDENTS);
      setDeliveries(DEMO_DELIVERIES);
      setSelectedProjectId("");
      setActiveIncidentId(null);
      setWorkspaceMenuOpen(false);
      setAuthRequired(true);
      setMode("demo");
      setConnectionNote("Signed out; exploring interactive demo orbit");
      setSigningOut(false);
      notify("Signed out of PockDev.");
    }
  };

  const handleSwitchWorkspace = async (workspace: WorkspaceSummary) => {
    setWorkspaceMenuOpen(false);
    if (workspace.id === me?.workspace?.id) return;
    const next: EngineSettings = {
      ...settings,
      workspaceId: workspace.id,
    };
    saveEngineSettings(next);
    setSettings(next);
    setSelectedProjectId("");
    setActiveIncidentId(null);
    const ok = await syncFromEngine(next, true);
    if (ok) {
      notify(`Switched workspace to ${workspace.name}.`);
    }
  };

  const handleCreateWorkspace = async (
    name: string
  ): Promise<WorkspaceSummary> => {
    const client = createEngineClient(settings);
    const created = await client.createWorkspace(name);
    const next: EngineSettings = {
      ...settings,
      workspaceId: created.id,
    };
    saveEngineSettings(next);
    setSettings(next);
    setSelectedProjectId("");
    setActiveIncidentId(null);
    await syncFromEngine(next, true);
    notify(`Created workspace ${created.name} and switched into it.`);
    return created;
  };

  const handleSaveEngineConfig = async (e: React.FormEvent) => {
    e.preventDefault();
    const next: EngineSettings = {
      baseUrl: draftUrl.trim() || DEFAULT_ENGINE_URL,
      token: draftToken.trim(),
      workspaceId: "",
    };
    saveEngineSettings(next);
    setSettings(next);
    const ok = await syncFromEngine(next, true);
    if (ok) {
      setConfigOpen(false);
      notify("Connected to live PockDev Go engine.");
    } else {
      notify("Could not connect to engine; check status note above.");
    }
  };

  const handleReplayDelivery = async (deliveryId: string) => {
    setReplayingId(deliveryId);
    try {
      if (mode === "live") {
        const client = createEngineClient(settings);
        const updated = await client.replayDelivery(deliveryId);
        setDeliveries((prev) =>
          prev.map((d) => (d.id === deliveryId ? updated : d))
        );
        const ov = await client.getOverview();
        setLiveOverview(ov);
        notify(`Queued ${deliveryId} to send again.`);
      } else {
        const result = applyDemoReplayDelivery(
          deliveries,
          incidents,
          deliveryId
        );
        setDeliveries(result.deliveries);
        setIncidents(result.incidents);
        notify(`Sent ${deliveryId} again -> HTTP 200 OK.`);
      }
    } catch (err) {
      notify(
        err instanceof Error ? err.message : "Could not send webhook again."
      );
    } finally {
      setReplayingId(null);
    }
  };

  const handleIncidentAction = async (
    incidentId: string,
    action: IncidentAction
  ) => {
    setBusyAction(action);
    setActionFeedback("");
    try {
      if (mode === "live") {
        const client = createEngineClient(settings);
        const updated = await client.runIncidentAction(incidentId, action);
        setIncidents((prev) =>
          prev.map((inc) => (inc.id === incidentId ? updated : inc))
        );
        const [ov, dlvs] = await Promise.all([
          client.getOverview(),
          client.listDeliveries(),
        ]);
        setLiveOverview(ov);
        setDeliveries(dlvs ?? []);
      } else {
        const result = applyDemoIncidentAction(
          incidents,
          deliveries,
          projects,
          incidentId,
          action
        );
        setIncidents(result.incidents);
        setDeliveries(result.deliveries);
      }

      const labelMap: Record<IncidentAction, string> = {
        draft_pr: "Drafted fix PR and updated incident state.",
        replay_all: "Sent all held events again.",
        resolve: "Marked resolved. All quiet.",
        mute: "Muted alerts for 1 hour.",
      };
      setActionFeedback(labelMap[action]);
      notify(labelMap[action]);
    } catch (err) {
      const msg =
        err instanceof Error ? err.message : "Action could not complete.";
      setActionFeedback(msg);
      notify(msg);
    } finally {
      setBusyAction(null);
    }
  };

  const handleUpgradeToPro = async () => {
    if (!settings.token.trim()) {
      notify("Sign in with GitHub first to upgrade your workspace.");
      return;
    }
    setUpgradingBilling(true);
    try {
      const client = createEngineClient(settings);
      const callbackUrl = `${window.location.origin}/deck?billing=verify`;
      const checkout = await client.startCheckout(callbackUrl);
      if (!checkout.authorizationUrl) {
        throw new Error("Paystack did not return a checkout URL.");
      }
      window.location.assign(checkout.authorizationUrl);
    } catch (err) {
      const msg =
        err instanceof Error
          ? err.message
          : "Could not start Paystack checkout.";
      notify(msg);
      setUpgradingBilling(false);
      throw err;
    }
  };

  const handleCreateProject = async (
    input: CreateProjectInput
  ): Promise<Project> => {
    if (mode === "live") {
      const client = createEngineClient(settings);
      const created = await client.createProject(input);
      setProjects((prev) => [created, ...prev]);
      const [ov, nextBilling] = await Promise.all([
        client.getOverview(),
        client.getBilling().catch(() => null),
      ]);
      setLiveOverview(ov);
      if (nextBilling) setBilling(nextBilling);
      notify(`Created shield for ${created.name}.`);
      return created;
    }

    const created = createDemoProject(input);
    setProjects((prev) => [...prev, created]);
    notify(`Created shield for ${created.name} (demo mode).`);
    return created;
  };

  const handleSimulateCrash = () => {
    const { incident, delivery } = createSimulatedDrill(projects);
    setIncidents((prev) => [incident, ...prev]);
    setDeliveries((prev) => [delivery, ...prev]);
    setActiveIncidentId(incident.id);
    notify("Simulated crash caught and linked to held webhooks.");
  };

  const showSignInBanner = mode === "demo" && (authRequired || !settings.token);

  return (
    <div className="mx-auto max-w-7xl px-5 pb-24 pt-7 sm:px-8">
      {/* Unauthenticated Geometric GitHub Sign-In Gate */}
      {showSignInBanner ? (
        <section id="signin" className="mb-6">
          <GeometricVessel
            shape="octagon"
            tone="citron"
            innerClassName="p-5 sm:p-7"
          >
            <div className="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
              <div className="flex items-start gap-4">
                <svg
                  width="54"
                  height="54"
                  viewBox="0 0 56 56"
                  className="hidden shrink-0 sm:block overflow-visible"
                  aria-hidden="true"
                >
                  <circle
                    cx="28"
                    cy="28"
                    r="24"
                    fill="none"
                    stroke="#C9F23D"
                    strokeOpacity="0.45"
                    strokeWidth="1.3"
                    strokeDasharray="10 7"
                    className="orbital-cw-slow"
                  />
                  <polygon
                    points="28,11 43,37 13,37"
                    fill="#1E241B"
                    stroke="#C9F23D"
                    strokeWidth="1.8"
                  />
                  <circle cx="28" cy="29" r="3.4" fill="#2AD4C5" />
                </svg>
                <div className="max-w-2xl">
                  <div className="flex flex-wrap items-center gap-2.5">
                    <span className="clip-hex-sm bg-citron px-3 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider text-silt">
                      Live Workspace Access
                    </span>
                    <span className="font-mono text-[11px] uppercase tracking-wider text-ash">
                      Demo constellation active below
                    </span>
                  </div>
                  <h2 className="mt-2 font-display text-xl font-extrabold text-chalk sm:text-2xl">
                    Sign in with GitHub to shield your real apps
                  </h2>
                  <p className="mt-1.5 text-sm leading-relaxed text-ash">
                    Connect your PockDev workspaces in one click to catch real
                    webhooks, isolate team orbits, and draft fix PRs right from
                    WhatsApp. You can also explore every replay and fix trigger
                    in demo mode below.
                  </p>
                  {githubAuthConfigured === false ? (
                    <p className="mt-2 font-mono text-xs text-cyan">
                      Note: this engine does not have GitHub OAuth configured
                      yet; use Advanced: Engine URL &amp; API Token to connect.
                    </p>
                  ) : null}
                </div>
              </div>

              <div className="flex flex-wrap items-center gap-3 shrink-0">
                <PrismButton
                  variant="citron"
                  size="md"
                  onClick={handleGitHubSignIn}
                >
                  <GitHubMark size={16} />
                  Sign in with GitHub
                </PrismButton>
                <PrismButton
                  variant="olive"
                  size="sm"
                  onClick={() => setConfigOpen((v) => !v)}
                >
                  {configOpen
                    ? "Hide advanced link"
                    : "Advanced: Engine URL & API Token"}
                </PrismButton>
              </div>
            </div>
          </GeometricVessel>
        </section>
      ) : null}

      {/* Top Engine Status, Workspace Switcher & Account Ribbon */}
      <GeometricVessel
        shape="octagon-sm"
        tone={mode === "live" ? "citron" : "cyan"}
        innerClassName="p-4 sm:p-5"
      >
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3.5">
            <svg width="24" height="24" viewBox="0 0 24 24" aria-hidden="true">
              <circle
                cx="12"
                cy="12"
                r="10"
                fill="none"
                stroke={mode === "live" ? "#C9F23D" : "#2AD4C5"}
                strokeWidth="1.4"
                strokeDasharray="6 4"
                className="orbital-cw-med"
              />
              <circle
                cx="12"
                cy="12"
                r="4"
                fill={mode === "live" ? "#C9F23D" : "#2AD4C5"}
              />
            </svg>
            <div>
              <div className="flex flex-wrap items-center gap-2.5">
                <h1 className="font-display text-lg font-extrabold text-chalk sm:text-xl">
                  Orbital Control Deck
                </h1>
                <span
                  className={`clip-hex-sm px-3 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider ${
                    mode === "live"
                      ? "bg-citron text-silt"
                      : "bg-cyan text-silt"
                  }`}
                >
                  {mode === "live"
                    ? `Live engine (${formatUptime(uptimeSeconds)})`
                    : mode === "connecting"
                      ? "Syncing"
                      : "Demo orbit"}
                </span>
              </div>
              <p className="mt-0.5 font-mono text-xs text-ash">
                {connectionNote}
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-2.5">
            {/* Signed-In Geometric Workspace Switcher */}
            {mode === "live" && me ? (
              <div className="relative" ref={workspaceMenuRef}>
                <button
                  type="button"
                  onClick={() => setWorkspaceMenuOpen((v) => !v)}
                  aria-expanded={workspaceMenuOpen}
                  aria-haspopup="menu"
                  className="clip-octagon-sm bg-cyan/80 p-[1px] transition-transform active:scale-[0.98]"
                >
                  <span className="flex items-center gap-2.5 clip-octagon-sm bg-silt px-3.5 py-1.5 text-left hover:bg-basalt">
                    <svg
                      width="12"
                      height="12"
                      viewBox="0 0 14 14"
                      aria-hidden="true"
                    >
                      <polygon
                        points="7,1 13,4.5 13,9.5 7,13 1,9.5 1,4.5"
                        fill="none"
                        stroke="#2AD4C5"
                        strokeWidth="1.5"
                      />
                    </svg>
                    <span>
                      <span className="block font-mono text-[9px] uppercase tracking-[0.18em] text-ash">
                        Workspace
                      </span>
                      <span className="block font-display text-xs font-bold text-chalk">
                        {me.workspace.name}
                      </span>
                    </span>
                    <svg
                      width="10"
                      height="10"
                      viewBox="0 0 10 10"
                      className={`ml-1 text-cyan transition-transform ${
                        workspaceMenuOpen ? "rotate-180" : ""
                      }`}
                      aria-hidden="true"
                    >
                      <polygon points="1,3 9,3 5,8" fill="currentColor" />
                    </svg>
                  </span>
                </button>

                {workspaceMenuOpen ? (
                  <div
                    role="menu"
                    className="absolute right-0 z-40 mt-2 w-72 shadow-2xl"
                  >
                    <GeometricVessel
                      shape="octagon-sm"
                      tone="cyan"
                      innerClassName="p-3 bg-silt"
                    >
                      <div className="border-b border-moss pb-2 px-1">
                        <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan">
                          Switch Workspace ({me.workspaces.length})
                        </span>
                      </div>
                      <div className="mt-2 space-y-1.5">
                        {me.workspaces.map((ws) => {
                          const isCurrent = ws.id === me.workspace.id;
                          return (
                            <button
                              key={ws.id}
                              type="button"
                              role="menuitem"
                              onClick={() => void handleSwitchWorkspace(ws)}
                              className={`w-full clip-prism-sm px-3.5 py-2 text-left transition-colors flex items-center justify-between gap-2 ${
                                isCurrent
                                  ? "bg-basalt text-citron"
                                  : "bg-olive/70 text-chalk hover:bg-basalt"
                              }`}
                            >
                              <span className="min-w-0">
                                <span className="block truncate font-display text-xs font-bold">
                                  {ws.name}
                                </span>
                                <span className="block truncate font-mono text-[10px] text-ash">
                                  {ws.slug}
                                </span>
                              </span>
                              <span className="clip-hex-sm bg-silt px-2.5 py-0.5 font-mono text-[9px] uppercase tracking-wider text-cyan shrink-0">
                                {ws.role || "member"}
                              </span>
                            </button>
                          );
                        })}
                      </div>
                      <div className="mt-3 border-t border-moss pt-2.5">
                        <PrismButton
                          variant="cyan"
                          size="sm"
                          className="w-full"
                          onClick={() => {
                            setWorkspaceMenuOpen(false);
                            setNewWorkspaceOpen(true);
                          }}
                        >
                          + New Workspace
                        </PrismButton>
                      </div>
                    </GeometricVessel>
                  </div>
                ) : null}
              </div>
            ) : null}

            {/* Signed-In GitHub User Identity & Sign Out */}
            {mode === "live" && me?.user ? (
              <div className="clip-octagon-sm bg-moss p-[1px]">
                <div className="flex items-center gap-2.5 clip-octagon-sm bg-silt px-3 py-1">
                  <OctagonAvatar
                    login={me.user.githubLogin}
                    avatarUrl={me.user.avatarUrl}
                    size={26}
                    tone="citron"
                  />
                  <span className="font-mono text-xs font-semibold text-chalk">
                    @{me.user.githubLogin}
                  </span>
                </div>
              </div>
            ) : mode === "live" && me?.mode === "token" ? (
              <span className="clip-hex-sm bg-basalt px-3 py-1 font-mono text-[10px] uppercase tracking-wider text-cyan">
                API Token
              </span>
            ) : null}

            {mode === "live" ? (
              <PrismButton
                variant="olive"
                size="sm"
                disabled={signingOut}
                onClick={() => void handleSignOut()}
              >
                {signingOut ? "Signing out..." : "Sign out"}
              </PrismButton>
            ) : null}

            {mode === "demo" ? (
              <PrismButton
                variant="terracotta"
                size="sm"
                onClick={handleSimulateCrash}
              >
                <svg
                  width="10"
                  height="10"
                  viewBox="0 0 12 12"
                  aria-hidden="true"
                >
                  <polygon points="6,1 11,10 1,10" fill="currentColor" />
                </svg>
                Simulate crash
              </PrismButton>
            ) : (
              <PrismButton
                variant="olive"
                size="sm"
                onClick={() => void syncFromEngine(settings)}
              >
                Refresh live
              </PrismButton>
            )}

            {!showSignInBanner ? (
              <PrismButton
                variant="olive"
                size="sm"
                onClick={() => setConfigOpen((v) => !v)}
              >
                {configOpen ? "Hide engine link" : "Engine link"}
              </PrismButton>
            ) : null}

            <PrismButton
              variant="citron"
              size="sm"
              onClick={() => setNewProjectOpen(true)}
            >
              + New Project
            </PrismButton>
          </div>
        </div>

        {/* Collapsible Advanced: Engine URL + Bearer Token Configuration */}
        {configOpen ? (
          <form
            onSubmit={handleSaveEngineConfig}
            className="mt-4 grid gap-3 border-t border-moss pt-4 sm:grid-cols-12 sm:items-end"
          >
            <div className="sm:col-span-12 flex flex-wrap items-center justify-between gap-2">
              <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-cyan">
                Advanced: Engine URL &amp; API Token (CLI and self-hosted)
              </span>
              <span className="font-mono text-[10px] text-ash">
                Saved locally in this browser
              </span>
            </div>

            <label className="sm:col-span-5 block">
              <span className="font-mono text-[10px] uppercase tracking-wider text-ash">
                Go Engine URL (NEXT_PUBLIC_ENGINE_URL)
              </span>
              <input
                type="url"
                value={draftUrl}
                onChange={(e) => setDraftUrl(e.target.value)}
                placeholder="http://localhost:8080"
                className="mt-1 w-full clip-octagon-sm bg-silt px-3.5 py-2 font-mono text-xs text-chalk border border-moss focus:border-citron focus:outline-none"
              />
            </label>

            <label className="sm:col-span-5 block">
              <span className="font-mono text-[10px] uppercase tracking-wider text-ash">
                API Bearer Token or Session Token
              </span>
              <input
                type="password"
                value={draftToken}
                onChange={(e) => setDraftToken(e.target.value)}
                placeholder="Bearer token from engine API_TOKEN"
                className="mt-1 w-full clip-octagon-sm bg-silt px-3.5 py-2 font-mono text-xs text-chalk border border-moss focus:border-citron focus:outline-none"
              />
            </label>

            <div className="sm:col-span-2 flex justify-end">
              <PrismButton type="submit" variant="cyan" size="sm">
                Connect
              </PrismButton>
            </div>
          </form>
        ) : null}
      </GeometricVessel>

      {/* Paystack Verification Banner */}
      {verifyingBilling ? (
        <div className="mt-5">
          <GeometricVessel
            shape="hex-bar"
            tone="cyan"
            innerClassName="px-6 py-3.5"
          >
            <div className="flex items-center gap-3">
              <svg
                width="18"
                height="18"
                viewBox="0 0 20 20"
                className="shrink-0 orbital-cw-med"
                aria-hidden="true"
              >
                <circle
                  cx="10"
                  cy="10"
                  r="8"
                  fill="none"
                  stroke="#2AD4C5"
                  strokeWidth="1.6"
                  strokeDasharray="12 6"
                />
              </svg>
              <p className="font-mono text-xs text-cyan">
                Verifying Paystack payment and activating Pro Orbit...
              </p>
            </div>
          </GeometricVessel>
        </div>
      ) : null}

      {/* Celebratory Pro Upgrade Toast */}
      {celebrationToast ? (
        <div className="mt-5" role="status" aria-live="polite">
          <GeometricVessel
            shape="hex-bar"
            tone="citron"
            innerClassName="bg-olive px-6 py-4"
          >
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div className="flex items-center gap-3.5">
                <svg
                  width="26"
                  height="26"
                  viewBox="0 0 28 28"
                  className="shrink-0"
                  aria-hidden="true"
                >
                  <circle
                    cx="14"
                    cy="14"
                    r="11"
                    fill="none"
                    stroke="#C9F23D"
                    strokeWidth="1.6"
                    strokeDasharray="7 3"
                  />
                  <polygon points="14,6 20,14 14,22 8,14" fill="#C9F23D" />
                </svg>
                <div>
                  <span className="clip-hex-sm inline-block bg-citron px-2.5 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.2em] text-silt">
                    PRO ORBIT UNLOCKED
                  </span>
                  <p className="mt-1 font-display text-sm font-bold text-chalk">
                    {celebrationToast}
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setCelebrationToast("")}
                className="clip-prism-sm bg-basalt px-3.5 py-1.5 font-mono text-[10px] uppercase tracking-wider text-chalk transition-colors hover:bg-citron hover:text-silt"
              >
                Dismiss
              </button>
            </div>
          </GeometricVessel>
        </div>
      ) : null}

      {/* Live Workspace Plan & Quota Bar */}
      {mode === "live" && billing ? (
        <section className="mt-5" aria-label="Workspace plan and quota usage">
          <QuotaBar
            billing={billing}
            upgrading={upgradingBilling}
            onUpgrade={() => {
              void handleUpgradeToPro().catch(() => {});
            }}
          />
        </section>
      ) : null}

      {/* 4 Circular SVG Orbital Telemetry Dials */}
      <section
        aria-label="Live reliability telemetry"
        className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4 border-y border-moss/60 py-4"
      >
        <OrbitalDial
          label="Watched Apps"
          value={String(overview.projectsCount)}
          subtext={
            overview.projectsCount === 1
              ? "1 shielded endpoint"
              : `${overview.projectsCount} shielded endpoints`
          }
          progress={Math.min(1, overview.projectsCount / 6)}
          color="#C9F23D"
        />
        <OrbitalDial
          label="Open Issues"
          value={String(overview.openIncidents)}
          subtext={
            overview.openIncidents === 0
              ? "All quiet"
              : `${overview.openIncidents} waiting for reply`
          }
          progress={
            overview.openIncidents === 0
              ? 1
              : Math.min(1, overview.openIncidents / 4)
          }
          color={overview.openIncidents > 0 ? "#E05A33" : "#C9F23D"}
          alert={overview.openIncidents > 0}
        />
        <OrbitalDial
          label="Safely Held"
          value={String(overview.pausedDeliveries)}
          subtext={
            overview.pausedDeliveries === 0
              ? "No paused webhooks"
              : `${overview.pausedDeliveries} events safely held`
          }
          progress={
            overview.pausedDeliveries === 0
              ? 0.08
              : Math.min(1, overview.pausedDeliveries / 12)
          }
          color={overview.pausedDeliveries > 0 ? "#E05A33" : "#2AD4C5"}
        />
        <OrbitalDial
          label="Healthy Rate"
          value={`${Math.round(overview.healthyRate)}%`}
          subtext={`${overview.caughtToday} caught today`}
          progress={overview.healthyRate / 100}
          color="#2AD4C5"
        />
      </section>

      {/* Inline status feedback strip */}
      {bannerNote ? (
        <div className="mt-4 clip-prism-sm bg-citron/15 border-l-2 border-citron px-4 py-2 font-mono text-xs text-citron">
          {bannerNote}
        </div>
      ) : null}

      {/* Radial Constellation of Projects + Active Incident Triangles */}
      <section className="mt-7">
        <OrbitalConstellation
          projects={projects}
          incidents={filteredIncidents}
          selectedProjectId={selectedProjectId}
          onSelectProject={setSelectedProjectId}
          onSelectIncident={(inc) => {
            setActionFeedback("");
            setActiveIncidentId(inc.id);
          }}
        />
      </section>

      {/* Breathing Orbital Project Selector Dock + Shield URL Inspector */}
      <section className="mt-8">
        <div className="flex flex-wrap items-center justify-between gap-2 mb-3">
          <span className="font-mono text-[11px] uppercase tracking-[0.2em] text-ash">
            Watched Apps Orbit
          </span>
          <span className="font-mono text-[11px] text-ash">
            Select an app ring to copy its Shield URL or filter its stream
          </span>
        </div>

        <div className="flex flex-wrap items-center gap-4">
          {projects.map((proj) => {
            const projInc = incidents.filter((i) => i.projectId === proj.id);
            const orbitState: OrbitState = projInc.some(
              (i) => i.status === "open"
            )
              ? "incident"
              : projInc.some((i) => i.status === "fixing")
                ? "fixing"
                : projInc.some((i) => i.status === "muted")
                  ? "muted"
                  : "healthy";
            const isSelected = selectedProjectId === proj.id;

            return (
              <button
                key={proj.id}
                type="button"
                onClick={() => setSelectedProjectId(isSelected ? "" : proj.id)}
                className={`clip-octagon-sm p-[1px] transition-transform hover:-translate-y-0.5 active:scale-[0.99] ${
                  isSelected
                    ? "bg-citron"
                    : orbitState === "incident"
                      ? "bg-terracotta/70"
                      : "bg-moss"
                }`}
              >
                <div className="clip-octagon-sm bg-olive px-3.5 py-2 flex items-center gap-3">
                  <OrbitalNode
                    state={orbitState}
                    size={54}
                    active={isSelected}
                  />
                  <div className="text-left pr-1">
                    <p className="font-display text-sm font-bold text-chalk">
                      {proj.name}
                    </p>
                    <p className="font-mono text-[10px] text-ash">
                      {orbitState === "incident"
                        ? "Needs attention"
                        : orbitState === "fixing"
                          ? "Fix PR drafted"
                          : orbitState === "muted"
                            ? "Muted 1h"
                            : "All quiet"}
                    </p>
                  </div>
                </div>
              </button>
            );
          })}
        </div>

        {selectedProject ? (
          <GeometricVessel
            shape="octagon-sm"
            tone="citron"
            className="mt-4"
            innerClassName="p-4 bg-silt"
          >
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div className="space-y-1 min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-display text-sm font-bold text-citron">
                    {selectedProject.name}
                  </span>
                  <span className="font-mono text-[11px] text-ash">
                    {selectedProject.githubRepo || "no GitHub repo linked"}
                  </span>
                </div>
                <p className="break-all font-mono text-xs text-chalk">
                  Shield URL:{" "}
                  <span className="text-cyan">
                    {settings.baseUrl.replace(/\/+$/, "")}/in/
                    {selectedProject.endpointKey}
                  </span>
                </p>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <PrismButton
                  variant="citron"
                  size="sm"
                  onClick={() =>
                    void copyToClipboard(
                      `${settings.baseUrl.replace(/\/+$/, "")}/in/${selectedProject.endpointKey}`,
                      "Shield URL"
                    )
                  }
                >
                  Copy Shield URL
                </PrismButton>
                <PrismButton
                  variant="cyan"
                  size="sm"
                  onClick={() =>
                    void copyToClipboard(selectedProject.sdkKey, "SDK Key")
                  }
                >
                  Copy SDK Key
                </PrismButton>
                <PrismButton
                  variant="olive"
                  size="sm"
                  onClick={() => setSelectedProjectId("")}
                >
                  Clear filter
                </PrismButton>
              </div>
            </div>
          </GeometricVessel>
        ) : null}
      </section>

      {/* Live Webhook River */}
      <WebhookRiver
        deliveries={filteredDeliveries}
        projects={projects}
        statusFilter={statusFilter}
        onStatusFilterChange={setStatusFilter}
        replayingId={replayingId}
        onReplay={handleReplayDelivery}
        onOpenIncident={(id) => {
          setActionFeedback("");
          setActiveIncidentId(id);
        }}
      />

      {/* Polygon Incident Drawer */}
      <IncidentDrawer
        incident={activeIncident}
        project={activeIncidentProject}
        busyAction={busyAction}
        actionFeedback={actionFeedback}
        onClose={() => setActiveIncidentId(null)}
        onAction={handleIncidentAction}
      />

      {/* Inline Polygonal New Project Modal */}
      <NewProjectModal
        open={newProjectOpen}
        engineUrl={settings.baseUrl}
        billing={mode === "live" ? billing : null}
        upgrading={upgradingBilling}
        onUpgrade={mode === "live" ? handleUpgradeToPro : undefined}
        onClose={() => setNewProjectOpen(false)}
        onCreate={handleCreateProject}
      />

      {/* Inline Polygonal New Workspace Modal */}
      <NewWorkspaceModal
        open={newWorkspaceOpen}
        onClose={() => setNewWorkspaceOpen(false)}
        onCreate={handleCreateWorkspace}
      />
    </div>
  );
}
