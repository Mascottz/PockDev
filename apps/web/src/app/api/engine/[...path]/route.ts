import { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

async function proxy(
  req: NextRequest,
  context: { params: { path: string[] } }
): Promise<Response> {
  const rawTarget = (
    req.headers.get("x-pockdev-target") ||
    process.env.NEXT_PUBLIC_ENGINE_URL ||
    ""
  )
    .trim()
    .replace(/\/+$/, "");

  if (!rawTarget.startsWith("https://") && !rawTarget.startsWith("http://")) {
    return Response.json(
      { error: "Engine URL must start with https:// or http://" },
      { status: 400 }
    );
  }

  const subPath = (context.params.path || []).join("/");
  const upstreamUrl = `${rawTarget}/${subPath}${req.nextUrl.search}`;

  const headers: Record<string, string> = {
    Accept: "application/json",
  };
  const auth = req.headers.get("authorization");
  if (auth) headers.Authorization = auth;
  const contentType = req.headers.get("content-type");
  if (contentType) headers["Content-Type"] = contentType;

  const hasBody = req.method !== "GET" && req.method !== "HEAD";
  const bodyText = hasBody ? await req.text() : undefined;

  try {
    const upstream = await fetch(upstreamUrl, {
      method: req.method,
      headers,
      body: bodyText && bodyText.length > 0 ? bodyText : undefined,
      cache: "no-store",
    });

    const text = await upstream.text();
    return new Response(text, {
      status: upstream.status,
      headers: {
        "Content-Type":
          upstream.headers.get("content-type") || "application/json; charset=utf-8",
        "Cache-Control": "no-store",
      },
    });
  } catch (err) {
    const detail = err instanceof Error ? err.message : "upstream unreachable";
    return Response.json(
      { error: `Could not reach engine at ${rawTarget}: ${detail}` },
      { status: 502 }
    );
  }
}

export async function GET(
  req: NextRequest,
  context: { params: { path: string[] } }
): Promise<Response> {
  return proxy(req, context);
}

export async function POST(
  req: NextRequest,
  context: { params: { path: string[] } }
): Promise<Response> {
  return proxy(req, context);
}
