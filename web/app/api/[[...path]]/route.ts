import { NextRequest, NextResponse } from "next/server";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

function backendBase(): string {
  const u = process.env.API_PROXY_URL?.trim() || "http://localhost:8080";
  return u.replace(/\/$/, "");
}

async function proxy(req: NextRequest, pathSegments: string[] | undefined) {
  const base = backendBase();
  if (!base) {
    return NextResponse.json(
      { error: "API_PROXY_URL is not configured" },
      { status: 503 }
    );
  }

  const suffix = pathSegments?.length ? pathSegments.join("/") : "";
  const apiPath = suffix ? `api/${suffix}` : "api";
  const target = new URL(apiPath, `${base}/`);
  target.search = req.nextUrl.searchParams.toString();

  const upstreamHeaders = new Headers();
  req.headers.forEach((value, key) => {
    const lower = key.toLowerCase();
    if (
      lower === "host" ||
      lower === "connection" ||
      lower === "content-length" ||
      lower === "transfer-encoding"
    ) {
      return;
    }
    upstreamHeaders.set(key, value);
  });

  const init: RequestInit = {
    method: req.method,
    headers: upstreamHeaders,
    redirect: "manual",
  };

  if (req.method !== "GET" && req.method !== "HEAD") {
    init.body = await req.arrayBuffer();
  }

  const res = await fetch(target.toString(), init);

  // Buffer the full body instead of streaming `res.body`. Forwarding hop-by-hop
  // headers (or a stale Content-Length after decode) can truncate the payload on
  // the browser side and break JSON.parse (e.g. "Unterminated string in JSON").
  const body = await res.arrayBuffer();

  const outHeaders = new Headers();
  const contentType = res.headers.get("content-type");
  if (contentType) {
    outHeaders.set("content-type", contentType);
  }

  return new NextResponse(body, {
    status: res.status,
    statusText: res.statusText,
    headers: outHeaders,
  });
}

type Ctx = { params: Promise<{ path?: string[] }> };

async function handle(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}

export const GET = handle;
export const POST = handle;
export const PUT = handle;
export const PATCH = handle;
export const DELETE = handle;
export const OPTIONS = handle;
