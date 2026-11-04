import { NextResponse } from "next/server";
import { checkRateLimit, getClientIp } from "@/lib/rate-limiter";
import type { components } from "@/types/api";

type SearchRequest = components["schemas"]["SearchRequest"];
type SearchResponse = components["schemas"]["SearchResponse"];
type ErrorResponse = components["schemas"]["ErrorResponse"];

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
  let body: SearchRequest;
  try {
    body = await req.json();
  } catch {
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

  if (!body || typeof body.query !== "string" || body.query.trim() === "") {
    return NextResponse.json<ErrorResponse>(
      {
        error: "validation_error",
        message: "Field 'query' is required and must not be empty.",
        code: 400,
      },
      {
        status: 400,
        headers: rateLimitHeaders,
      }
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
      body: JSON.stringify(body),
      cache: "no-store",
    });

    const data = await backendRes.json();

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

    return NextResponse.json(data, {
      status: backendRes.status,
      headers: responseHeaders,
    });
  } catch (error) {
    const msg = error instanceof Error ? error.message : "Unknown connection failure";
    return NextResponse.json<ErrorResponse>(
      {
        error: "backend_unavailable",
        message: `Failed to connect to search backend at ${backendBaseUrl}: ${msg}`,
        code: 502,
      },
      {
        status: 502,
        headers: rateLimitHeaders,
      }
    );
  }
}
