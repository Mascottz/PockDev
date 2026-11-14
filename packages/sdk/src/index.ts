export interface PockDevConfig {
  key: string;
  endpoint?: string;
}

export interface ErrorPayload {
  message: string;
  stack: string;
  filePath: string;
  line: number;
  level: "error" | "fatal";
  deliveryId: string;
  context: Record<string, unknown>;
}

const DELIVERY_HEADER = "x-pockdev-delivery-id";
const FRAME_RE = /(?:\(|at\s+|@)((?:file:\/\/)?(?:[A-Za-z]:)?[^\s()]+?):(\d+)(?::\d+)?\)?/;

let activeKey = "";
let activeEndpoint = "http://localhost:8080";

/** Configure @pockdev/sdk once at app startup. */
export function init(config: PockDevConfig): void {
  activeKey = config.key.trim();
  activeEndpoint = (config.endpoint || "http://localhost:8080").trim().replace(/\/+$/, "");
}

/** Extract the first relevant file path and line number from a stack trace. */
export function parseTopFrame(stack: string): { filePath: string; line: number } {
  let fallback: { filePath: string; line: number } | null = null;
  for (const rawLine of stack.split("\n").slice(1)) {
    const match = FRAME_RE.exec(rawLine);
    if (!match) continue;
    const filePath = match[1].replace(/^file:\/\//, "");
    const frame = { filePath, line: Number.parseInt(match[2], 10) || 0 };
    if (!fallback) fallback = frame;
    if (!filePath.includes("node:internal") && !filePath.includes("@pockdev/sdk")) return frame;
  }
  return fallback ?? { filePath: "", line: 0 };
}

/** Read x-pockdev-delivery-id from Node, Express, Fastify, Hono, or Next.js requests. */
export function readDeliveryId(req: unknown): string {
  if (!req || typeof req !== "object") return "";
  const candidate = req as Record<string, unknown>;
  const honoReq = candidate.req as Record<string, unknown> | undefined;
  if (honoReq && typeof honoReq.header === "function") {
    const val = (honoReq.header as (name: string) => string | undefined)(DELIVERY_HEADER);
    if (val) return val.trim();
  }
  const rawReq = (honoReq?.raw ?? candidate.raw ?? candidate) as Record<string, unknown>;
  const headers = (candidate.headers ?? rawReq.headers) as Record<string, unknown> | undefined;
  if (!headers) return "";
  if (typeof (headers as { get?: unknown }).get === "function") {
    return ((headers as { get: (name: string) => string | null }).get(DELIVERY_HEADER) ?? "").trim();
  }
  for (const [k, v] of Object.entries(headers)) {
    if (k.toLowerCase() !== DELIVERY_HEADER) continue;
    return (Array.isArray(v) ? String(v[0] ?? "") : typeof v === "string" ? v : "").trim();
  }
  return "";
}

/** Capture an error and send it to the PockDev engine without throwing. */
export async function capture(
  err: unknown,
  req?: unknown,
  extraContext?: Record<string, unknown>
): Promise<void> {
  if (!activeKey) return;
  const error = err instanceof Error ? err : new Error(typeof err === "string" ? err : String(err));
  const stack = error.stack ?? `${error.name}: ${error.message}`;
  const { filePath, line } = parseTopFrame(stack);
  const level: "error" | "fatal" = extraContext?.level === "fatal" ? "fatal" : "error";
  const context: Record<string, unknown> = { ...(extraContext ?? {}) };
  delete context.level;
  const payload: ErrorPayload = {
    message: error.message || "Unhandled error",
    stack,
    filePath,
    line,
    level,
    deliveryId: readDeliveryId(req),
    context,
  };
  try {
    const signal = typeof AbortSignal !== "undefined" && "timeout" in AbortSignal
      ? AbortSignal.timeout(3000)
      : undefined;
    await fetch(`${activeEndpoint}/v1/errors`, {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-PockDev-Key": activeKey },
      body: JSON.stringify(payload),
      signal,
    });
  } catch {
    // Reporting must never interrupt the host application.
  }
}

/** Wrap a Node HTTP, Express, Fastify, Hono, or Next.js route handler to report crashes. */
export function wrap<T extends (...args: any[]) => any>(handler: T): T {
  const wrapped = (...args: Parameters<T>): ReturnType<T> => {
    const req = args[0];
    const next = args.length >= 3 && typeof args[2] === "function" ? args[2] : null;
    try {
      const result = handler(...args);
      if (result && typeof result.then === "function") {
        return result.catch((err: unknown) =>
          capture(err, req).then(() => {
            if (next) return next(err);
            throw err;
          })
        ) as ReturnType<T>;
      }
      return result;
    } catch (err) {
      void capture(err, req);
      if (next) return next(err);
      throw err;
    }
  };
  return wrapped as T;
}
