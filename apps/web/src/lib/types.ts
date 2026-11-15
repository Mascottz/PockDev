export type DeliveryStatus = "delivered" | "retrying" | "paused" | "failed";

export type IncidentStatus = "open" | "fixing" | "resolved" | "muted";

export type IncidentAction = "draft_pr" | "replay_all" | "resolve" | "mute";

export interface HealthResponse {
  ok: boolean;
  uptimeSeconds: number;
  githubAuthConfigured?: boolean;
}

export interface AuthUser {
  id: string;
  githubLogin: string;
  name: string;
  email: string;
  avatarUrl: string;
}

export interface WorkspaceSummary {
  id: string;
  name: string;
  slug: string;
  role: string;
}

export interface MeResponse {
  mode: "session" | "token";
  user: AuthUser | null;
  workspace: {
    id: string;
    name: string;
    slug: string;
  };
  workspaces: WorkspaceSummary[];
}

export interface Overview {
  projectsCount: number;
  openIncidents: number;
  pausedDeliveries: number;
  caughtToday: number;
  healthyRate: number;
}

export interface Project {
  id: string;
  name: string;
  slug: string;
  endpointKey: string;
  sdkKey: string;
  targetUrl: string;
  githubRepo: string;
  whatsappPhone: string;
  status: string;
  createdAt: string;
}

export interface CreateProjectInput {
  name: string;
  slug: string;
  targetUrl: string;
  githubRepo: string;
  whatsappPhone: string;
}

export interface Delivery {
  id: string;
  projectId: string;
  status: DeliveryStatus;
  httpStatus: number;
  attemptCount: number;
  schemaDriftNote: string;
  payloadPreview: string;
  linkedIncidentId: string;
  createdAt: string;
}

export interface Incident {
  id: string;
  projectId: string;
  title: string;
  plainSummary: string;
  filePath: string;
  commitSha: string;
  status: IncidentStatus;
  pausedCount: number;
  prUrl: string;
  lastSeenAt: string;
}

export interface EngineSettings {
  baseUrl: string;
  token: string;
  workspaceId?: string;
}

export interface BillingUsage {
  projectsUsed: number;
  projectsLimit: number;
  monthlyEventsUsed: number;
  monthlyEventsLimit: number;
  retentionDays: number;
}

export interface BillingStatus {
  plan: "free" | "pro";
  planStatus: string;
  planExpiresAt: string | null;
  paystackConfigured: boolean;
  currency: string;
  proPriceKobo: number;
  usage: BillingUsage;
}

export interface CheckoutResponse {
  authorizationUrl: string;
  reference: string;
}
