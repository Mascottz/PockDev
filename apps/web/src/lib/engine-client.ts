import type {
  AuthUser,
  CreateProjectInput,
  Delivery,
  EngineSettings,
  HealthResponse,
  Incident,
  IncidentAction,
  MeResponse,
  Overview,
  Project,
  WorkspaceSummary,
} from "./types";

export const DEFAULT_ENGINE_URL =
  process.env.NEXT_PUBLIC_ENGINE_URL || "http://localhost:8080";

export const AUTH_CHANGED_EVENT = "pockdev-auth-changed";

const STORAGE_KEY = "pockdev_engine_settings_v1";
const USER_STORAGE_KEY = "pockdev_auth_user_v1";

function dispatchAuthChanged(): void {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new Event(AUTH_CHANGED_EVENT));
}

export function loadEngineSettings(): EngineSettings {
  if (typeof window === "undefined") {
    return { baseUrl: DEFAULT_ENGINE_URL, token: "", workspaceId: "" };
  }
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return { baseUrl: DEFAULT_ENGINE_URL, token: "", workspaceId: "" };
    }
    const parsed = JSON.parse(raw) as Partial<EngineSettings>;
    return {
      baseUrl: parsed.baseUrl?.trim() || DEFAULT_ENGINE_URL,
      token: parsed.token?.trim() || "",
      workspaceId: parsed.workspaceId?.trim() || "",
    };
  } catch {
    return { baseUrl: DEFAULT_ENGINE_URL, token: "", workspaceId: "" };
  }
}

export function saveEngineSettings(settings: EngineSettings): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        baseUrl: settings.baseUrl.trim() || DEFAULT_ENGINE_URL,
        token: settings.token.trim(),
        workspaceId: settings.workspaceId?.trim() || "",
      })
    );
  } catch {
    // Ignore storage quota errors in restricted environments.
  }
  dispatchAuthChanged();
}

export function loadCachedAuthUser(): AuthUser | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(USER_STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<AuthUser>;
    if (!parsed.githubLogin) return null;
    return {
      id: parsed.id || "",
      githubLogin: parsed.githubLogin,
      name: parsed.name || "",
      email: parsed.email || "",
      avatarUrl: parsed.avatarUrl || "",
    };
  } catch {
    return null;
  }
}

export function saveCachedAuthUser(user: AuthUser | null): void {
  if (typeof window === "undefined") return;
  try {
    if (!user) {
      window.localStorage.removeItem(USER_STORAGE_KEY);
    } else {
      window.localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(user));
    }
  } catch {
    // Ignore storage errors in restricted environments.
  }
  dispatchAuthChanged();
}

export function clearAuthSession(): EngineSettings {
  const current = loadEngineSettings();
  const cleared: EngineSettings = {
    baseUrl: current.baseUrl,
    token: "",
    workspaceId: "",
  };
  if (typeof window !== "undefined") {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(cleared));
      window.localStorage.removeItem(USER_STORAGE_KEY);
    } catch {
      // Ignore storage errors in restricted environments.
    }
    dispatchAuthChanged();
  }
  return cleared;
}

function normalizeBase(url: string): string {
  return (url.trim() || DEFAULT_ENGINE_URL).replace(/\/+$/, "");
}

export function buildGitHubLoginUrl(baseUrl: string, origin: string): string {
  const cleanOrigin = origin.trim().replace(/\/+$/, "");
  return `${normalizeBase(baseUrl)}/auth/github/start?redirect_to=${encodeURIComponent(
    `${cleanOrigin}/auth/callback`
  )}`;
}

async function requestJson<T>(
  settings: EngineSettings,
  path: string,
  options: {
    method?: string;
    body?: unknown;
    auth?: boolean;
    timeoutMs?: number;
  } = {}
): Promise<T> {
  const { method = "GET", body, auth = true, timeoutMs = 12000 } = options;
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  const headers: Record<string, string> = {
    Accept: "application/json",
  };
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (auth && settings.token.trim()) {
    headers.Authorization = `Bearer ${settings.token.trim()}`;
  }
  if (auth && settings.workspaceId?.trim()) {
    headers["X-PockDev-Workspace"] = settings.workspaceId.trim();
  }

  const base = normalizeBase(settings.baseUrl);
  const isLocalTarget =
    base.startsWith("http://localhost") || base.startsWith("http://127.0.0.1");
  const requestUrl =
    typeof window !== "undefined" && !isLocalTarget
      ? `/api/engine${path}`
      : `${base}${path}`;
  if (typeof window !== "undefined" && !isLocalTarget) {
    headers["X-PockDev-Target"] = base;
  }

  try {
    const response = await fetch(requestUrl, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      signal: controller.signal,
    });

    if (!response.ok) {
      let detail = `Engine returned ${response.status}`;
      try {
        const errJson = (await response.json()) as { error?: string };
        if (errJson.error) detail = errJson.error;
      } catch {
        // Keep status message fallback.
      }
      throw new Error(detail);
    }

    return (await response.json()) as T;
  } finally {
    clearTimeout(timer);
  }
}

export function createEngineClient(settings: EngineSettings) {
  return {
    getHealth: () =>
      requestJson<HealthResponse>(settings, "/healthz", {
        auth: false,
        timeoutMs: 10000,
      }),

    getMe: () => requestJson<MeResponse>(settings, "/api/me"),

    createWorkspace: (name: string) =>
      requestJson<WorkspaceSummary>(settings, "/api/workspaces", {
        method: "POST",
        body: { name },
      }),

    logout: () =>
      requestJson<{ ok: boolean }>(settings, "/api/auth/logout", {
        method: "POST",
      }),

    getOverview: () => requestJson<Overview>(settings, "/api/overview"),

    listProjects: () => requestJson<Project[]>(settings, "/api/projects"),

    createProject: (input: CreateProjectInput) =>
      requestJson<Project>(settings, "/api/projects", {
        method: "POST",
        body: input,
      }),

    listDeliveries: (params?: { projectId?: string; status?: string }) => {
      const search = new URLSearchParams();
      if (params?.projectId) search.set("projectId", params.projectId);
      if (params?.status) search.set("status", params.status);
      const qs = search.toString();
      return requestJson<Delivery[]>(
        settings,
        `/api/deliveries${qs ? `?${qs}` : ""}`
      );
    },

    replayDelivery: (id: string) =>
      requestJson<Delivery>(
        settings,
        `/api/deliveries/${encodeURIComponent(id)}/replay`,
        {
          method: "POST",
        }
      ),

    listIncidents: (projectId?: string) => {
      const qs = projectId ? `?projectId=${encodeURIComponent(projectId)}` : "";
      return requestJson<Incident[]>(settings, `/api/incidents${qs}`);
    },

    runIncidentAction: (id: string, action: IncidentAction) =>
      requestJson<Incident>(
        settings,
        `/api/incidents/${encodeURIComponent(id)}/action`,
        {
          method: "POST",
          body: { action },
        }
      ),
  };
}
