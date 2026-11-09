import { NextResponse } from "next/server";
import { checkRateLimit, getClientIp } from "@/lib/rate-limiter";
import { sanitizeSearchRequest } from "@/lib/sanitize-request";
import type { components } from "@/types/api";

type SearchResponse = components["schemas"]["SearchResponse"];
type ErrorResponse = components["schemas"]["ErrorResponse"];

export const maxDuration = 45;

const MAX_REQUEST_BYTES = 1 << 20;

async function readBoundedBody(req: Request): Promise<unknown> {
  const declaredLength = Number(req.headers.get("content-length"));
  if (Number.isFinite(declaredLength) && declaredLength > MAX_REQUEST_BYTES) {
    throw new RangeError("request body too large");
  }

  if (!req.body) return {};
  const reader = req.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    total += value.byteLength;
    if (total > MAX_REQUEST_BYTES) {
      await reader.cancel();
      throw new RangeError("request body too large");
    }
    chunks.push(value);
  }

  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return JSON.parse(new TextDecoder().decode(bytes));
}

export async function POST(req: Request): Promise<NextResponse<SearchResponse | ErrorResponse>> {
  // 1. Enforce per-IP rate limiting at the route handler level
  const ip = getClientIp(req);
  const rateLimitMax = parseInt(process.env.PLEWTUS_RATE_LIMIT_PER_MINUTE || "20", 10);
  const rateLimit = checkRateLimit(ip, rateLimitMax);

  const rateLimitHeaders = {
    "X-RateLimit-Limit": String(rateLimit.limit),
    "X-RateLimit-Remaining": String(rateLimit.remaining),
    "X-RateLimit-Reset": String(rateLimit.reset),
  };

  if (!rateLimit.success) {
    return NextResponse.json<ErrorResponse>(
      {
        error: "rate_limit_exceeded",
        message: `Frontend rate limit exceeded (${rateLimit.limit} req/min). Please retry in ${rateLimit.reset}s.`,
        code: 429,
      },
      {
        status: 429,
        headers: {
          ...rateLimitHeaders,
          "Retry-After": String(rateLimit.reset),
        },
      }
    );
  }

  // 2. Validate server-side API key configuration.
  // CRITICAL: If the server-side key is missing, never contact the backend.
  const apiKey = process.env.PLEWTUS_API_KEY;
  if (!apiKey || apiKey.trim() === "") {
    return NextResponse.json<ErrorResponse>(
      {
        error: "configuration_error",
        message: "Server API key is not configured. Backend cannot be contacted.",
        code: 500,
      },
      {
        status: 500,
        headers: rateLimitHeaders,
      }
    );
  }

  // 3. Parse and validate request body
  let rawBody: unknown;
  try {
    rawBody = await readBoundedBody(req);
  } catch (error) {
    if (error instanceof RangeError) {
      return NextResponse.json<ErrorResponse>(
        { error: "payload_too_large", message: `Request body exceeds ${MAX_REQUEST_BYTES} bytes.`, code: 413 },
        { status: 413, headers: rateLimitHeaders }
      );
    }
    return NextResponse.json<ErrorResponse>(
      {
        error: "invalid_json",
        message: "Malformed JSON in request payload.",
        code: 400,
      },
      {
        status: 400,
        headers: rateLimitHeaders,
      }
    );
  }

  const sanitized = sanitizeSearchRequest(rawBody);
  if (!sanitized.ok) {
    return NextResponse.json<ErrorResponse>(
      { error: "validation_error", message: sanitized.message, code: 400 },
      { status: 400, headers: rateLimitHeaders }
    );
  }

  // 4. Dispatch request to the Go backend with server-side credentials
  const backendBaseUrl = process.env.PLEWTUS_BACKEND_URL || "http://localhost:8080";
  const targetUrl = `${backendBaseUrl.replace(/\/+$/, "")}/search`;

  try {
    const backendRes = await fetch(targetUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${apiKey}`,
      },
      body: JSON.stringify(sanitized.request),
      cache: "no-store",
      signal: AbortSignal.timeout(45_000),
    });

    let data: unknown;
    try {
      data = JSON.parse(await backendRes.text());
    } catch {
      return NextResponse.json<ErrorResponse>(
        { error: "invalid_backend_response", message: "Search backend returned a non-JSON response.", code: 502 },
        { status: 502, headers: rateLimitHeaders }
      );
    }

    // Relay backend quota headers if present, alongside our rate limit headers
    const responseHeaders: Record<string, string> = {
      ...rateLimitHeaders,
    };

    const quotaLimit = backendRes.headers.get("X-Quota-Limit");
    const quotaRemaining = backendRes.headers.get("X-Quota-Remaining");
    const quotaReset = backendRes.headers.get("X-Quota-Reset");
    if (quotaLimit) responseHeaders["X-Quota-Limit"] = quotaLimit;
    if (quotaRemaining) responseHeaders["X-Quota-Remaining"] = quotaRemaining;
    if (quotaReset) responseHeaders["X-Quota-Reset"] = quotaReset;
    const retryAfter = backendRes.headers.get("Retry-After");
    if (retryAfter) responseHeaders["Retry-After"] = retryAfter;

    return NextResponse.json<SearchResponse | ErrorResponse>(data as SearchResponse | ErrorResponse, {
      status: backendRes.status,
      headers: responseHeaders,
    });
  } catch (error) {
    // Log details server-side; don't reveal internal hostnames to visitors.
    console.error("search backend request failed", error);
    const timedOut = error instanceof Error && error.name === "TimeoutError";
    return NextResponse.json<ErrorResponse>(
      {
        error: timedOut ? "backend_timeout" : "backend_unavailable",
        message: timedOut
          ? "The search backend took too long to respond. Please retry."
          : "The search backend is unreachable right now. Please retry shortly.",
        code: 502,
      },
      {
        status: 502,
        headers: rateLimitHeaders,
      }
    );
  }
}
