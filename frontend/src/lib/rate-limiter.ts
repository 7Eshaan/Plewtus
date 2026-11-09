import { isIP as isNodeIP } from "node:net";

export interface RateLimitResult {
  success: boolean;
  limit: number;
  remaining: number;
  reset: number; // seconds until oldest entry expires or window resets
}

const MAX_TRACKED_IPS = 5000;

class InMemoryRateLimiter {
  private requests: Map<string, number[]> = new Map();
  private readonly windowMs: number;
  private readonly defaultLimit: number;
  private lastCleanup: number = Date.now();

  constructor(windowMs: number = 60_000, defaultLimit: number = 20) {
    this.windowMs = windowMs;
    this.defaultLimit = defaultLimit;
  }

  public check(ip: string, limit?: number): RateLimitResult {
    const now = Date.now();
    const effectiveLimit = limit && limit > 0 ? limit : this.defaultLimit;
    const windowStart = now - this.windowMs;

    this.maybeCleanup(now);

    let timestamps = this.requests.get(ip);
    if (!timestamps) {
      // Keep high-cardinality forwarded headers from growing process memory forever.
      if (this.requests.size >= MAX_TRACKED_IPS) {
        const oldestKey = this.requests.keys().next().value;
        if (oldestKey !== undefined) this.requests.delete(oldestKey);
      }
      timestamps = [];
      this.requests.set(ip, timestamps);
    }

    // Retain only requests within the sliding window
    timestamps = timestamps.filter((t) => t > windowStart);
    this.requests.delete(ip);
    this.requests.set(ip, timestamps);

    if (timestamps.length >= effectiveLimit) {
      const oldest = timestamps[0] ?? now;
      const reset = Math.max(1, Math.ceil((oldest + this.windowMs - now) / 1000));
      return {
        success: false,
        limit: effectiveLimit,
        remaining: 0,
        reset,
      };
    }

    timestamps.push(now);
    const remaining = Math.max(0, effectiveLimit - timestamps.length);
    const reset = Math.max(1, Math.ceil((timestamps[0] + this.windowMs - now) / 1000));

    return {
      success: true,
      limit: effectiveLimit,
      remaining,
      reset,
    };
  }

  // Clear state for tests or maintenance
  public reset(): void {
    this.requests.clear();
  }

  private maybeCleanup(now: number): void {
    // Run cleanup at most once per 60 seconds
    if (now - this.lastCleanup < 60_000) return;
    this.lastCleanup = now;

    const windowStart = now - this.windowMs;
    for (const [ip, timestamps] of this.requests.entries()) {
      const active = timestamps.filter((t) => t > windowStart);
      if (active.length === 0) {
        this.requests.delete(ip);
      } else {
        this.requests.set(ip, active);
      }
    }
  }
}

// Global singleton instance for the route handler
const globalLimiter = new InMemoryRateLimiter(60_000, 20);

export function checkRateLimit(ip: string, limit?: number): RateLimitResult {
  return globalLimiter.check(ip, limit);
}

export function resetRateLimits(): void {
  globalLimiter.reset();
}

/**
 * Identifies the client for rate limiting.
 *
 * Forwarding headers are only trusted when a proxy we control sets them:
 * on Vercel (the platform overwrites `x-vercel-forwarded-for`) or when
 * PLEWTUS_TRUST_PROXY_HEADERS=true behind a reverse proxy that appends the
 * client address to X-Forwarded-For. Otherwise any caller could rotate the
 * header to bypass the limit, so all untrusted traffic shares one bucket.
 */
export function getClientIp(req: Request): string {
  if (process.env.VERCEL) {
    const vercelIp = req.headers.get("x-vercel-forwarded-for")?.split(",")[0]?.trim();
    if (vercelIp && isNodeIP(vercelIp)) return vercelIp;
  }

  if (process.env.PLEWTUS_TRUST_PROXY_HEADERS === "true") {
    const realIp = req.headers.get("x-real-ip")?.trim();
    if (realIp && isNodeIP(realIp)) return realIp;
    // The right-most entry is the one appended by our own proxy.
    const hops = (req.headers.get("x-forwarded-for") ?? "")
      .split(",")
      .map((h) => h.trim())
      .filter(Boolean);
    const clientIp = hops[hops.length - 1];
    if (clientIp && isNodeIP(clientIp)) return clientIp;
  }

  return "untrusted-shared";
}
