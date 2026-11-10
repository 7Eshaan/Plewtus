import { isIP } from "node:net";

/**
 * Serves site icons for result cards from our own origin. Without this the
 * visitor's browser would request every result's icon from a third party,
 * disclosing which sites their searches returned along with their IP address.
 */

const HOSTNAME = /^(?=.{1,253}$)([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$/;
const MAX_ICON_BYTES = 100 * 1024;

function notFound(): Response {
  return new Response(null, { status: 404, headers: { "Cache-Control": "public, max-age=3600" } });
}

export async function GET(request: Request): Promise<Response> {
  const domain = (new URL(request.url).searchParams.get("domain") ?? "").trim().toLowerCase();
  // Only plain public hostnames are forwarded: no IPs, ports, paths, or credentials.
  if (!HOSTNAME.test(domain) || isIP(domain) !== 0) return notFound();

  try {
    const upstream = await fetch(`https://www.google.com/s2/favicons?domain=${encodeURIComponent(domain)}&sz=32`, {
      signal: AbortSignal.timeout(4000),
      cache: "no-store",
    });
    const type = upstream.headers.get("content-type") ?? "";
    if (!upstream.ok || !type.startsWith("image/")) return notFound();

    const body = await upstream.arrayBuffer();
    if (body.byteLength === 0 || body.byteLength > MAX_ICON_BYTES) return notFound();

    return new Response(body, {
      headers: {
        "Content-Type": type,
        "Cache-Control": "public, max-age=604800, immutable",
        "X-Content-Type-Options": "nosniff",
      },
    });
  } catch {
    return notFound();
  }
}
