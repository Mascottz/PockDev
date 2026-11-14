"use client";

import type { Incident, Project } from "@/lib/types";
import { GeometricVessel } from "./GeometricVessel";
import { TriangleBeacon } from "./TriangleBeacon";

interface OrbitalConstellationProps {
  projects: Project[];
  incidents: Incident[];
  selectedProjectId: string;
  onSelectProject: (projectId: string) => void;
  onSelectIncident: (incident: Incident) => void;
}

function hostLabel(url: string): string {
  try {
    return new URL(url).host;
  } catch {
    return url || "no target set";
  }
}

export function OrbitalConstellation({
  projects,
  incidents,
  selectedProjectId,
  onSelectProject,
  onSelectIncident,
}: OrbitalConstellationProps) {
  const cx = 310;
  const cy = 225;
  const orbitR = 148;

  const openOrFixing = incidents.filter(
    (inc) => inc.status === "open" || inc.status === "fixing"
  );

  // Place projects evenly on the orbital ring.
  const projectPositions = projects.map((project, index) => {
    const angle =
      -Math.PI / 2 + (index * Math.PI * 2) / Math.max(projects.length, 1);
    const x = cx + Math.cos(angle) * orbitR;
    const y = cy + Math.sin(angle) * orbitR;
    const projectIncidents = incidents.filter((i) => i.projectId === project.id);
    const openInc = projectIncidents.find((i) => i.status === "open");
    const fixingInc = projectIncidents.find((i) => i.status === "fixing");
    const state = openInc
      ? ("incident" as const)
      : fixingInc
        ? ("fixing" as const)
        : ("healthy" as const);
    return { project, x, y, angle, state, projectIncidents };
  });

  return (
    <div className="grid gap-6 lg:grid-cols-12">
      {/* Radial SVG Orbital Constellation */}
      <div className="lg:col-span-7">
        <GeometricVessel
          shape="octagon"
          tone={openOrFixing.length > 0 ? "terracotta" : "citron"}
          innerClassName="p-4 sm:p-6"
        >
          <div className="flex flex-wrap items-center justify-between gap-2 pb-3 border-b border-moss/60">
            <div>
              <span className="font-mono text-[11px] uppercase tracking-[0.2em] text-ash">
                Radial Constellation
              </span>
              <h2 className="font-display text-lg font-bold text-chalk">
                Watched apps and active issue beacons
              </h2>
            </div>
            {selectedProjectId ? (
              <button
                type="button"
                onClick={() => onSelectProject("")}
                className="clip-prism-sm bg-basalt px-3.5 py-1 font-mono text-[11px] uppercase tracking-wider text-citron hover:bg-moss"
              >
                Show all apps
              </button>
            ) : (
              <span className="clip-hex-sm bg-basalt px-3.5 py-1 font-mono text-[11px] uppercase tracking-wider text-ash">
                Click any circle or triangle
              </span>
            )}
          </div>

          <div className="relative mt-2 overflow-hidden">
            <svg
              viewBox="0 0 620 450"
              className="mx-auto h-auto w-full max-w-[620px]"
              role="img"
              aria-label="Radial constellation of PockDev watched apps and incident triangles"
            >
              {/* Concentric orbital guide circles */}
              <circle
                cx={cx}
                cy={cy}
                r="205"
                fill="none"
                stroke="#2C3527"
                strokeOpacity="0.4"
                strokeWidth="1"
                strokeDasharray="3 9"
              />
              <circle
                cx={cx}
                cy={cy}
                r={orbitR}
                fill="none"
                stroke="#2C3527"
                strokeOpacity="0.7"
                strokeWidth="1.2"
              />
              <circle
                cx={cx}
                cy={cy}
                r="88"
                fill="none"
                stroke="#2C3527"
                strokeOpacity="0.45"
                strokeWidth="1"
                strokeDasharray="6 8"
              />

              {/* Animated SVG flow conduits from Center Shield -> Each Project */}
              {projectPositions.map(({ project, x, y, state }) => {
                const selected = selectedProjectId === project.id;
                const stroke =
                  state === "incident"
                    ? "#E05A33"
                    : state === "fixing"
                      ? "#2AD4C5"
                      : "#C9F23D";
                return (
                  <g key={`link-${project.id}`}>
                    <line
                      x1={cx}
                      y1={cy}
                      x2={x}
                      y2={y}
                      stroke="#2C3527"
                      strokeWidth={selected ? 2.5 : 1.5}
                    />
                    <line
                      x1={cx}
                      y1={cy}
                      x2={x}
                      y2={y}
                      stroke={stroke}
                      strokeOpacity={selected || !selectedProjectId ? 0.9 : 0.35}
                      strokeWidth={selected ? 2.6 : 1.8}
                      className={
                        state === "incident"
                          ? "flow-conduit-alert"
                          : state === "fixing"
                            ? "flow-conduit-active"
                            : "flow-conduit-calm"
                      }
                    />
                  </g>
                );
              })}

              {/* Center PockDev Shield Core */}
              <g transform={`translate(${cx}, ${cy})`}>
                <circle
                  cx="0"
                  cy="0"
                  r="48"
                  fill="none"
                  stroke={openOrFixing.length > 0 ? "#E05A33" : "#C9F23D"}
                  strokeOpacity="0.38"
                  strokeWidth="1.2"
                  strokeDasharray="8 6"
                  className="orbital-breath"
                />
                <g className="orbital-cw-slow">
                  <circle
                    cx="0"
                    cy="0"
                    r="39"
                    fill="none"
                    stroke="#2AD4C5"
                    strokeOpacity="0.65"
                    strokeWidth="1.4"
                    strokeDasharray="24 14"
                  />
                  <circle cx="0" cy="-39" r="3" fill="#2AD4C5" />
                </g>
                <circle
                  cx="0"
                  cy="0"
                  r="31"
                  fill="#0C0E0B"
                  stroke={openOrFixing.length > 0 ? "#E05A33" : "#C9F23D"}
                  strokeWidth="2"
                />
                <text
                  x="0"
                  y="-4"
                  textAnchor="middle"
                  fill="#F3EFE6"
                  fontSize="11"
                  fontWeight="800"
                >
                  SHIELD
                </text>
                <text
                  x="0"
                  y="11"
                  textAnchor="middle"
                  fill={openOrFixing.length > 0 ? "#E05A33" : "#C9F23D"}
                  fontSize="9"
                  fontFamily="monospace"
                >
                  {openOrFixing.length > 0
                    ? `${openOrFixing.length} ALERT`
                    : "ALL QUIET"}
                </text>
              </g>

              {/* Project Breathing Circular Nodes + Orbiting Incident Triangles */}
              {projectPositions.map(
                ({ project, x, y, angle, state, projectIncidents }) => {
                  const selected = selectedProjectId === project.id;
                  const ringColor =
                    state === "incident"
                      ? "#E05A33"
                      : state === "fixing"
                        ? "#2AD4C5"
                        : "#C9F23D";
                  const activeInc = projectIncidents.find(
                    (i) => i.status !== "resolved"
                  );

                  // Place triangle beacon slightly outward from project node
                  const triDist = 60;
                  const tx = x + Math.cos(angle + 0.36) * triDist;
                  const ty = y + Math.sin(angle + 0.36) * triDist;

                  return (
                    <g key={project.id}>
                      {/* Tether to active incident triangle */}
                      {activeInc ? (
                        <line
                          x1={x}
                          y1={y}
                          x2={tx}
                          y2={ty}
                          stroke={
                            activeInc.status === "fixing" ? "#2AD4C5" : "#E05A33"
                          }
                          strokeWidth="1.8"
                          strokeDasharray="4 4"
                        />
                      ) : null}

                      {/* Clickable Project Circle */}
                      <g
                        transform={`translate(${x}, ${y})`}
                        onClick={() =>
                          onSelectProject(selected ? "" : project.id)
                        }
                        className="cursor-pointer"
                      >
                        <circle
                          cx="0"
                          cy="0"
                          r="38"
                          fill="none"
                          stroke={ringColor}
                          strokeOpacity={selected ? 0.85 : 0.3}
                          strokeWidth={selected ? 1.8 : 1}
                          strokeDasharray="4 6"
                          className="orbital-breath"
                        />
                        <g
                          className={
                            state === "incident"
                              ? "orbital-cw-fast"
                              : state === "fixing"
                                ? "orbital-cw-med"
                                : "orbital-cw-slow"
                          }
                        >
                          <circle
                            cx="0"
                            cy="0"
                            r="29"
                            fill="none"
                            stroke={ringColor}
                            strokeOpacity="0.7"
                            strokeWidth="1.4"
                          />
                          <circle cx="0" cy="-29" r="3.4" fill={ringColor} />
                        </g>
                        <circle
                          cx="0"
                          cy="0"
                          r="22"
                          fill={selected ? "#1E241B" : "#151913"}
                          stroke={ringColor}
                          strokeWidth={selected ? 2.6 : 1.8}
                        />
                        <text
                          x="0"
                          y="4"
                          textAnchor="middle"
                          fill="#F3EFE6"
                          fontSize="11"
                          fontWeight="700"
                          fontFamily="monospace"
                        >
                          {project.slug.slice(0, 3).toUpperCase()}
                        </text>

                        {/* Project Nameplate */}
                        <text
                          x="0"
                          y="51"
                          textAnchor="middle"
                          fill={selected ? "#C9F23D" : "#F3EFE6"}
                          fontSize="11.5"
                          fontWeight="700"
                        >
                          {project.name}
                        </text>
                        <text
                          x="0"
                          y="64"
                          textAnchor="middle"
                          fill="#A5A196"
                          fontSize="9"
                          fontFamily="monospace"
                        >
                          {hostLabel(project.targetUrl)}
                        </text>
                      </g>

                      {/* Clickable Pulsing Incident Triangle Beacon */}
                      {activeInc ? (
                        <g
                          transform={`translate(${tx}, ${ty})`}
                          onClick={() => onSelectIncident(activeInc)}
                          className="cursor-pointer"
                        >
                          {activeInc.status === "open" ? (
                            <polygon
                              points="0,-24 22,15 -22,15"
                              fill="none"
                              stroke="#E05A33"
                              strokeWidth="1.4"
                              className="beacon-shockwave"
                            />
                          ) : null}
                          <polygon
                            points="0,-20 18,12 -18,12"
                            fill={
                              activeInc.status === "fixing"
                                ? "#0F2624"
                                : activeInc.status === "muted"
                                  ? "#1E241B"
                                  : "#31160F"
                            }
                            stroke={
                              activeInc.status === "fixing"
                                ? "#2AD4C5"
                                : activeInc.status === "muted"
                                  ? "#A5A196"
                                  : "#E05A33"
                            }
                            strokeWidth="2.2"
                          />
                          <text
                            x="0"
                            y="6"
                            textAnchor="middle"
                            fill="#F3EFE6"
                            fontSize="10"
                            fontWeight="800"
                            fontFamily="monospace"
                          >
                            {activeInc.pausedCount > 0
                              ? activeInc.pausedCount
                              : "!"}
                          </text>
                        </g>
                      ) : null}
                    </g>
                  );
                }
              )}
            </svg>
          </div>
        </GeometricVessel>
      </div>

      {/* Active Incident Beacons Column */}
      <div className="lg:col-span-5 flex flex-col gap-3.5">
        <div className="flex items-center justify-between px-1">
          <div>
            <span className="font-mono text-[11px] uppercase tracking-[0.2em] text-terracotta">
              Triangular Beacons
            </span>
            <h2 className="font-display text-lg font-bold text-chalk">
              Active issues ({openOrFixing.length})
            </h2>
          </div>
          <span className="font-mono text-[11px] text-ash">
            {incidents.length} total
          </span>
        </div>

        {incidents.length === 0 ? (
          <GeometricVessel
            shape="octagon-sm"
            tone="citron"
            innerClassName="p-7 text-center"
          >
            <svg
              width="48"
              height="48"
              viewBox="0 0 64 64"
              className="mx-auto"
              aria-hidden="true"
            >
              <circle
                cx="32"
                cy="32"
                r="24"
                fill="none"
                stroke="#C9F23D"
                strokeWidth="2"
              />
              <polyline
                points="21,33 29,41 44,24"
                fill="none"
                stroke="#C9F23D"
                strokeWidth="2.5"
              />
            </svg>
            <p className="mt-3 font-display text-lg font-bold text-citron">
              All quiet
            </p>
            <p className="mt-1 text-sm text-ash">
              Every watched app is delivering webhooks cleanly.
            </p>
          </GeometricVessel>
        ) : (
          incidents.map((incident) => {
            const proj = projects.find((p) => p.id === incident.projectId);
            const tone =
              incident.status === "open"
                ? "terracotta"
                : incident.status === "fixing"
                  ? "cyan"
                  : incident.status === "resolved"
                    ? "citron"
                    : "moss";
            const statusText =
              incident.status === "open"
                ? "Open issue"
                : incident.status === "fixing"
                  ? "Fix PR open"
                  : incident.status === "muted"
                    ? "Muted 1h"
                    : "All quiet";

            return (
              <button
                key={incident.id}
                type="button"
                onClick={() => onSelectIncident(incident)}
                className="w-full text-left transition-transform duration-150 hover:-translate-y-0.5 active:scale-[0.99]"
              >
                <GeometricVessel
                  shape="octagon-sm"
                  tone={tone}
                  innerClassName="p-4 hover:bg-basalt/70 transition-colors"
                >
                  <div className="flex items-start gap-3.5">
                    <TriangleBeacon status={incident.status} size={42} />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <span className="font-mono text-[11px] uppercase tracking-wider text-ash">
                          {proj?.name ?? incident.projectId}
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
                          {statusText}
                        </span>
                      </div>

                      <p className="mt-1.5 font-display text-sm font-bold leading-snug text-chalk">
                        {incident.title}
                      </p>

                      <div className="mt-2.5 flex flex-wrap items-center gap-3 font-mono text-[11px]">
                        <span className="text-cyan">
                          {incident.filePath || "webhook dead letter"}
                        </span>
                        <span className="text-terracotta">
                          {incident.pausedCount > 0
                            ? `${incident.pausedCount} ${
                                incident.pausedCount === 1 ? "event" : "events"
                              } safely held`
                            : "0 held"}
                        </span>
                      </div>
                    </div>
                  </div>
                </GeometricVessel>
              </button>
            );
          })
        )}
      </div>
    </div>
  );
}
