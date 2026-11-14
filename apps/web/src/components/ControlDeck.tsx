"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
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
  createEngineClient,
  DEFAULT_ENGINE_URL,
  loadEngineSettings,
  saveEngineSettings,
} from "@/lib/engine-client";
import type {
  CreateProjectInput,
  Delivery,
  EngineSettings,
  Incident,
  IncidentAction,
  Overview,
  Project,
} from "@/lib/types";
import { GeometricVessel } from "./GeometricVessel";
import { IncidentDrawer } from "./IncidentDrawer";
import { NewProjectModal } from "./NewProjectModal";
import { OrbitalConstellation } from "./OrbitalConstellation";
import { OrbitalNode, type OrbitState } from "./OrbitalNode";
import { PrismButton } from "./PrismButton";
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
  });
  const [draftUrl, setDraftUrl] = useState(DEFAULT_ENGINE_URL);
  const [draftToken, setDraftToken] = useState("");
  const [configOpen, setConfigOpen] = useState(false);

  const [mode, setMode] = useState<ConnectionMode>("connecting");
  const [uptimeSeconds, setUptimeSeconds] = useState(0);
  const [connectionNote, setConnectionNote] = useState(
    "Checking PockDev Go engine..."
  );

  const [projects, setProjects] = useState<Project[]>(DEMO_PROJECTS);
  const [incidents, setIncidents] = useState<Incident[]>(DEMO_INCIDENTS);
  const [deliveries, setDeliveries] = useState<Delivery[]>(DEMO_DELIVERIES);
  const [liveOverview, setLiveOverview] = useState<Overview | null>(null);

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

  const syncFromEngine = useCallback(async (cfg: EngineSettings) => {
    setMode("connecting");
    const client = createEngineClient(cfg);
    try {
      const health = await client.getHealth();
      if (!health.ok) throw new Error("Engine reported unhealthy status");
      setUptimeSeconds(health.uptimeSeconds);

      const [ov, prjs, incs, dlvs] = await Promise.all([
        client.getOverview(),
        client.listProjects(),
        client.listIncidents(),
        client.listDeliveries(),
      ]);

      setLiveOverview(ov);
      setProjects(prjs ?? []);
      setIncidents(incs ?? []);
      setDeliveries(dlvs ?? []);
      setMode("live");
      setConnectionNote(`Engine live; up ${formatUptime(health.uptimeSeconds)}`);
    } catch (err) {
      setLiveOverview(null);
      setMode("demo");
      const msg = err instanceof Error ? err.message : "";
      if (msg.toLowerCase().includes("token") || msg.includes("401")) {
        setConnectionNote(
          "Engine reachable; add Bearer token in Engine link (using interactive demo)"
        );
      } else {
        setConnectionNote(
          "Interactive demo mode; connect Go engine anytime via Engine link"
        );
      }
    }
  }, []);

  useEffect(() => {
    const saved = loadEngineSettings();
    setSettings(saved);
    setDraftUrl(saved.baseUrl);
    setDraftToken(saved.token);
    void syncFromEngine(saved);
  }, [syncFromEngine]);

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

  const handleSaveEngineConfig = async (e: React.FormEvent) => {
    e.preventDefault();
    const next: EngineSettings = {
      baseUrl: draftUrl.trim() || DEFAULT_ENGINE_URL,
      token: draftToken.trim(),
    };
    saveEngineSettings(next);
    setSettings(next);
    setConfigOpen(false);
    await syncFromEngine(next);
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
        const result = applyDemoReplayDelivery(deliveries, incidents, deliveryId);
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

  const handleCreateProject = async (
    input: CreateProjectInput
  ): Promise<Project> => {
    if (mode === "live") {
      const client = createEngineClient(settings);
      const created = await client.createProject(input);
      setProjects((prev) => [created, ...prev]);
      const ov = await client.getOverview();
      setLiveOverview(ov);
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

  return (
    <div className="mx-auto max-w-7xl px-5 pb-24 pt-7 sm:px-8">
      {/* Top Engine Status & Control Ribbon */}
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
            {mode === "demo" ? (
              <PrismButton
                variant="terracotta"
                size="sm"
                onClick={handleSimulateCrash}
              >
                <svg width="10" height="10" viewBox="0 0 12 12" aria-hidden="true">
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

            <PrismButton
              variant="olive"
              size="sm"
              onClick={() => setConfigOpen((v) => !v)}
            >
              {configOpen ? "Hide engine link" : "Engine link"}
            </PrismButton>

            <PrismButton
              variant="citron"
              size="sm"
              onClick={() => setNewProjectOpen(true)}
            >
              + New Project
            </PrismButton>
          </div>
        </div>

        {/* Collapsible Engine URL + Bearer Token Configuration */}
        {configOpen ? (
          <form
            onSubmit={handleSaveEngineConfig}
            className="mt-4 grid gap-3 border-t border-moss pt-4 sm:grid-cols-12 sm:items-end"
          >
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
                API Bearer Token
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
          progress={overview.openIncidents === 0 ? 1 : Math.min(1, overview.openIncidents / 4)}
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
                onClick={() =>
                  setSelectedProjectId(isSelected ? "" : proj.id)
                }
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
        onClose={() => setNewProjectOpen(false)}
        onCreate={handleCreateProject}
      />
    </div>
  );
}
